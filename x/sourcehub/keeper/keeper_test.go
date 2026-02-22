package keeper

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"testing"
	"time"

	"cosmossdk.io/log"
	cosmosstore "cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/shinzonetwork/shinzohub/x/sourcehub/types"
)

// ─── Mock ICA Keeper ─────────────────────────────────────────────────────────

// mockICAKeeper satisfies types.ICAControllerKeeper without requiring real IBC
// infrastructure.
type mockICAKeeper struct {
	icaAddress string
	icaFound   bool
	sendTxErr  error
}

func (m *mockICAKeeper) RegisterInterchainAccount(
	_ sdk.Context,
	_, _, _ string,
	_ channeltypes.Order,
) error {
	return nil
}

func (m *mockICAKeeper) SendTx(
	_ sdk.Context,
	_, _ string,
	_ icatypes.InterchainAccountPacketData,
	_ uint64,
) (uint64, error) {
	return 1, m.sendTxErr
}

func (m *mockICAKeeper) GetInterchainAccountAddress(
	_ sdk.Context,
	_, _ string,
) (string, bool) {
	return m.icaAddress, m.icaFound
}

// ─── Test Fixture ─────────────────────────────────────────────────────────────

type testFixture struct {
	k         Keeper
	ctx       sdk.Context
	msgSrv    types.MsgServer
	authority string
	ica       *mockICAKeeper
}

// newTestFixture creates an isolated keeper backed by an in-memory store.  Each
// call returns a fresh, independent fixture so sub-tests cannot interfere.
func newTestFixture(t *testing.T) *testFixture {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := cosmosstore.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	storeService := runtime.NewKVStoreService(storeKey)

	// Use the governance module address as the default authority – the same
	// convention used by DefaultGenesis.
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	ctx := sdk.NewContext(ms, cmtproto.Header{
		ChainID: "test-chain",
		Height:  1,
		Time:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}, false, log.NewNopLogger())

	ica := &mockICAKeeper{}
	k := NewKeeper(cdc, storeService, ica, authority)

	// Bootstrap params so IsAdmin can read them without an error.
	k.SetParams(ctx, types.Params{Admin: authority})

	return &testFixture{
		k:         k,
		ctx:       ctx,
		msgSrv:    NewMsgServerImpl(k),
		authority: authority,
		ica:       ica,
	}
}

// testDelegateAddr returns a deterministic, valid bech32 address for use as
// DelegateAddress in attestation messages.
func testDelegateAddr(seed byte) string {
	addr := make([]byte, 20)
	addr[0] = seed
	return sdk.AccAddress(addr).String()
}

// testICAAddr returns a valid bech32 address to represent a remote ICA account.
func testICAAddr() string {
	return authtypes.NewModuleAddress("ica-account").String()
}

// ─── Indexer Attestation – KV Layer ──────────────────────────────────────────

func Test_IndexerAttestation_SetGet(t *testing.T) {
	delegate := testDelegateAddr(0xDE)

	t.Run("stores and retrieves attestation by delegate+chain key", func(t *testing.T) {
		f := newTestFixture(t)
		att := types.IndexerAttestation{
			ConsensusPubKey: "consensus-pub-key-001",
			DelegateAddress: delegate,
			SourceChain:     "ethereum",
			SourceChainId:   1,
			AttestationId:   "att-001",
		}

		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, att))

		got, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "ethereum", 1)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, att.ConsensusPubKey, got.ConsensusPubKey)
		require.Equal(t, att.DelegateAddress, got.DelegateAddress)
		require.Equal(t, att.SourceChain, got.SourceChain)
		require.EqualValues(t, att.SourceChainId, got.SourceChainId)
		require.Equal(t, att.AttestationId, got.AttestationId)
	})

	t.Run("returns not-found for an unknown delegate+chain combination", func(t *testing.T) {
		f := newTestFixture(t)
		_, found, err := f.k.GetIndexerAttestation(f.ctx, testDelegateAddr(0xFF), "ethereum", 1)
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("re-attesting the same delegate+chain overwrites the record (key rotation)", func(t *testing.T) {
		f := newTestFixture(t)
		first := types.IndexerAttestation{
			ConsensusPubKey: "pub-key-v1",
			DelegateAddress: delegate,
			SourceChain:     "ethereum",
			SourceChainId:   1,
			AttestationId:   "att-v1",
		}
		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, first))

		// Same chain – this is a key rotation / update.
		updated := types.IndexerAttestation{
			ConsensusPubKey: "pub-key-v2",
			DelegateAddress: delegate,
			SourceChain:     "ethereum",
			SourceChainId:   1,
			AttestationId:   "att-v2",
		}
		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, updated))

		got, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "ethereum", 1)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "pub-key-v2", got.ConsensusPubKey)
		require.Equal(t, "att-v2", got.AttestationId)
	})

	t.Run("same delegate attested for multiple chains coexist independently", func(t *testing.T) {
		f := newTestFixture(t)

		ethAtt := types.IndexerAttestation{
			ConsensusPubKey: "key-eth",
			DelegateAddress: delegate,
			SourceChain:     "ethereum",
			SourceChainId:   1,
			AttestationId:   "att-eth",
		}
		polyAtt := types.IndexerAttestation{
			ConsensusPubKey: "key-poly",
			DelegateAddress: delegate,
			SourceChain:     "polygon",
			SourceChainId:   137,
			AttestationId:   "att-poly",
		}
		baseAtt := types.IndexerAttestation{
			ConsensusPubKey: "key-base",
			DelegateAddress: delegate,
			SourceChain:     "base",
			SourceChainId:   8453,
			AttestationId:   "att-base",
		}

		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, ethAtt))
		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, polyAtt))
		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, baseAtt))

		gotEth, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "ethereum", 1)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "key-eth", gotEth.ConsensusPubKey)

		gotPoly, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "polygon", 137)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "key-poly", gotPoly.ConsensusPubKey)

		gotBase, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "base", 8453)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "key-base", gotBase.ConsensusPubKey)

		// A wrong chain lookup returns not-found.
		_, found, err = f.k.GetIndexerAttestation(f.ctx, delegate, "optimism", 10)
		require.NoError(t, err)
		require.False(t, found)
	})

	t.Run("chains sharing a numeric ID are kept distinct by chain name", func(t *testing.T) {
		f := newTestFixture(t)
		// Two different testnets that both use chainId 1337.
		alphaAtt := types.IndexerAttestation{
			ConsensusPubKey: "key-alpha",
			DelegateAddress: delegate,
			SourceChain:     "alpha-testnet",
			SourceChainId:   1337,
			AttestationId:   "att-alpha",
		}
		betaAtt := types.IndexerAttestation{
			ConsensusPubKey: "key-beta",
			DelegateAddress: delegate,
			SourceChain:     "beta-testnet",
			SourceChainId:   1337,
			AttestationId:   "att-beta",
		}

		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, alphaAtt))
		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, betaAtt))

		gotAlpha, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "alpha-testnet", 1337)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "key-alpha", gotAlpha.ConsensusPubKey)

		gotBeta, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "beta-testnet", 1337)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "key-beta", gotBeta.ConsensusPubKey)
	})

	t.Run("distinct delegate addresses are stored independently", func(t *testing.T) {
		f := newTestFixture(t)
		addrA := testDelegateAddr(0x01)
		addrB := testDelegateAddr(0x02)

		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, types.IndexerAttestation{
			ConsensusPubKey: "key-A",
			DelegateAddress: addrA,
			SourceChain:     "ethereum",
			SourceChainId:   1,
			AttestationId:   "att-A",
		}))
		require.NoError(t, f.k.SetIndexerAttestation(f.ctx, types.IndexerAttestation{
			ConsensusPubKey: "key-B",
			DelegateAddress: addrB,
			SourceChain:     "ethereum",
			SourceChainId:   1,
			AttestationId:   "att-B",
		}))

		gotA, foundA, err := f.k.GetIndexerAttestation(f.ctx, addrA, "ethereum", 1)
		require.NoError(t, err)
		require.True(t, foundA)
		require.Equal(t, "key-A", gotA.ConsensusPubKey)

		gotB, foundB, err := f.k.GetIndexerAttestation(f.ctx, addrB, "ethereum", 1)
		require.NoError(t, err)
		require.True(t, foundB)
		require.Equal(t, "key-B", gotB.ConsensusPubKey)
	})
}

// ─── Admin Checks ─────────────────────────────────────────────────────────────

func Test_Keeper_IsAdmin(t *testing.T) {
	f := newTestFixture(t)

	customAdmin := sdk.AccAddress(make([]byte, 20))
	customAdmin[0] = 0xCA
	customAdminStr := customAdmin.String()

	nonAdmin := sdk.AccAddress(make([]byte, 20))
	nonAdmin[0] = 0xBB
	nonAdminStr := nonAdmin.String()

	t.Run("module authority is always an admin", func(t *testing.T) {
		require.True(t, f.k.IsAdmin(f.ctx, f.authority))
	})

	t.Run("custom admin from params is also an admin", func(t *testing.T) {
		f.k.SetParams(f.ctx, types.Params{Admin: customAdminStr})
		require.True(t, f.k.IsAdmin(f.ctx, customAdminStr))
		// Authority must still be recognised regardless of params.
		require.True(t, f.k.IsAdmin(f.ctx, f.authority))
	})

	t.Run("arbitrary address is not an admin", func(t *testing.T) {
		require.False(t, f.k.IsAdmin(f.ctx, nonAdminStr))
	})
}

// ─── MsgServer – AddIndexerAttestation ───────────────────────────────────────

func Test_MsgServer_AddIndexerAttestation(t *testing.T) {
	// Generate a real secp256k1 key pair for the delegate.  The keeper's
	// verifyDelegateSignature performs ecrecover, so DelegateAddress must be
	// the bech32 encoding of the Ethereum address derived from this key.
	delegateKey, err := ethcrypto.GenerateKey()
	require.NoError(t, err)
	delegateEthAddr := ethcrypto.PubkeyToAddress(delegateKey.PublicKey)
	delegate := sdk.AccAddress(delegateEthAddr.Bytes()).String()

	digest := sha256.Sum256([]byte("test-attestation-payload"))
	sig, err := ethcrypto.Sign(digest[:], delegateKey)
	require.NoError(t, err)

	// buildMsg constructs a well-formed attestation message with a valid
	// delegate ownership proof.
	buildMsg := func(signer, sourceChain string, sourceChainId uint64, consensusPubKey, attestationId string) *types.MsgIndexerAttestation {
		return &types.MsgIndexerAttestation{
			Signer:            signer,
			ConsensusPubKey:   consensusPubKey,
			DelegateAddress:   delegate,
			SourceChain:       sourceChain,
			SourceChainId:     sourceChainId,
			AttestationId:     attestationId,
			DelegateDigest:    digest[:],
			DelegateSignature: sig,
		}
	}

	t.Run("admin successfully attests an indexer", func(t *testing.T) {
		f := newTestFixture(t)
		msg := buildMsg(f.authority, "ethereum", 1, "consensus-pub-key-abc", "att-0001")

		resp, err := f.msgSrv.AddIndexerAttestation(f.ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// State must be persisted under the composite (delegate, chain) key.
		got, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "ethereum", 1)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, msg.ConsensusPubKey, got.ConsensusPubKey)
		require.Equal(t, msg.SourceChain, got.SourceChain)
		require.EqualValues(t, msg.SourceChainId, got.SourceChainId)
		require.Equal(t, msg.AttestationId, got.AttestationId)

		// An IndexerAttested event must have been emitted with the correct
		// attributes.
		events := f.ctx.EventManager().Events()
		require.NotEmpty(t, events)
		var eventFound bool
		for _, ev := range events {
			if ev.Type != "IndexerAttested" {
				continue
			}
			eventFound = true
			attrs := make(map[string]string, len(ev.Attributes))
			for _, a := range ev.Attributes {
				attrs[a.Key] = a.Value
			}
			require.Equal(t, f.authority, attrs["signer"])
			require.Equal(t, msg.ConsensusPubKey, attrs["consensus_pub_key"])
			require.Equal(t, delegate, attrs["delegate_address"])
			require.Equal(t, "ethereum", attrs["source_chain"])
			require.Equal(t, "1", attrs["source_chain_id"])
			require.Equal(t, "att-0001", attrs["attestation_id"])
		}
		require.True(t, eventFound, "IndexerAttested event was not emitted")
	})

	t.Run("non-admin signer is rejected with ErrUnauthorized", func(t *testing.T) {
		f := newTestFixture(t)
		nonAdmin := sdk.AccAddress(make([]byte, 20))
		nonAdmin[0] = 0x99
		// Admin check fires before signature check; sig contents don't matter here.
		msg := buildMsg(nonAdmin.String(), "ethereum", 1, "key", "att-x")

		_, err := f.msgSrv.AddIndexerAttestation(f.ctx, msg)
		require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	})

	t.Run("mismatched delegate signature is rejected with ErrUnauthorized", func(t *testing.T) {
		f := newTestFixture(t)

		// Generate a different key – its address won't match delegate.
		wrongKey, err := ethcrypto.GenerateKey()
		require.NoError(t, err)
		wrongSig, err := ethcrypto.Sign(digest[:], wrongKey)
		require.NoError(t, err)

		msg := &types.MsgIndexerAttestation{
			Signer:            f.authority,
			ConsensusPubKey:   "key",
			DelegateAddress:   delegate,
			SourceChain:       "ethereum",
			SourceChainId:     1,
			AttestationId:     "att-wrong",
			DelegateDigest:    digest[:],
			DelegateSignature: wrongSig,
		}
		_, err = f.msgSrv.AddIndexerAttestation(f.ctx, msg)
		require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	})

	t.Run("re-attesting the same delegate+chain updates the record", func(t *testing.T) {
		f := newTestFixture(t)

		_, err := f.msgSrv.AddIndexerAttestation(f.ctx, buildMsg(f.authority, "ethereum", 1, "consensus-pub-key-abc", "att-0001"))
		require.NoError(t, err)

		// Same chain – acts as a key rotation.
		_, err = f.msgSrv.AddIndexerAttestation(f.ctx, buildMsg(f.authority, "ethereum", 1, "new-consensus-key", "att-0002"))
		require.NoError(t, err)

		got, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "ethereum", 1)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "new-consensus-key", got.ConsensusPubKey)
		require.Equal(t, "att-0002", got.AttestationId)
	})

	t.Run("attesting the same delegate for a second chain creates an independent record", func(t *testing.T) {
		f := newTestFixture(t)

		// First attestation: ethereum.
		_, err := f.msgSrv.AddIndexerAttestation(f.ctx, buildMsg(f.authority, "ethereum", 1, "consensus-pub-key-abc", "att-0001"))
		require.NoError(t, err)

		// Second attestation: polygon – must NOT overwrite the ethereum entry.
		_, err = f.msgSrv.AddIndexerAttestation(f.ctx, buildMsg(f.authority, "polygon", 137, "poly-consensus-key", "att-poly"))
		require.NoError(t, err)

		// Both records must be independently retrievable.
		gotEth, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "ethereum", 1)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "consensus-pub-key-abc", gotEth.ConsensusPubKey)

		gotPoly, found, err := f.k.GetIndexerAttestation(f.ctx, delegate, "polygon", 137)
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, "poly-consensus-key", gotPoly.ConsensusPubKey)
	})
}

// ─── RegisterEntity – Precondition Errors ─────────────────────────────────────

// Test_RegisterEntity_Preconditions verifies that RegisterEntity returns
// descriptive errors before reaching the ACP/ICA layer when required module
// state is missing or signatures are invalid.
func Test_RegisterEntity_Preconditions(t *testing.T) {
	// Generate key material shared across sub-tests so each one can reach a
	// different failure point without repeating setup.
	peerPub, peerPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	nodePriv, err := secp256k1.GeneratePrivateKey()
	require.NoError(t, err)
	nodePub := nodePriv.PubKey().SerializeCompressed()

	message := []byte("register-entity-test-message")
	peerSig := ed25519.Sign(peerPriv, message)

	msgHash := sha256.Sum256(message)
	nodeSig := ecdsa.Sign(nodePriv, msgHash[:]).Serialize()

	signerAddr := make([]byte, 20)
	signerAddr[0] = 0xAB

	t.Run("fails when controller connection ID is not set", func(t *testing.T) {
		f := newTestFixture(t)
		// Connection ID is intentionally left unset.

		_, _, err := f.k.RegisterEntity(
			f.ctx,
			peerPub, peerSig,
			nodePub, nodeSig,
			message,
			types.RoleIndexer,
			signerAddr,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no connection ID set")
	})

	t.Run("fails when ICA address is not found for the connection", func(t *testing.T) {
		f := newTestFixture(t)
		f.k.SetControllerConnectionID(f.ctx, "connection-0")
		// mockICAKeeper.icaFound defaults to false – simulates a missing ICA.

		_, _, err := f.k.RegisterEntity(
			f.ctx,
			peerPub, peerSig,
			nodePub, nodeSig,
			message,
			types.RoleIndexer,
			signerAddr,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "ICA address not found")
	})

	t.Run("fails when policy ID is not set", func(t *testing.T) {
		f := newTestFixture(t)
		f.k.SetControllerConnectionID(f.ctx, "connection-0")
		f.ica.icaAddress = testICAAddr()
		f.ica.icaFound = true
		// Policy ID intentionally left unset in the store.

		_, _, err := f.k.RegisterEntity(
			f.ctx,
			peerPub, peerSig,
			nodePub, nodeSig,
			message,
			types.RoleIndexer,
			signerAddr,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no policy ID set")
	})

	t.Run("fails when peer key (ed25519) signature does not match", func(t *testing.T) {
		f := newTestFixture(t)
		f.k.SetControllerConnectionID(f.ctx, "connection-0")
		f.k.SetPolicyId(f.ctx, "policy-abc")
		f.ica.icaAddress = testICAAddr()
		f.ica.icaFound = true

		// All-zero signature of the correct length is cryptographically invalid.
		badPeerSig := make([]byte, ed25519.SignatureSize)

		_, _, err := f.k.RegisterEntity(
			f.ctx,
			peerPub, badPeerSig,
			nodePub, nodeSig,
			message,
			types.RoleIndexer,
			signerAddr,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid peer key signature")
	})

	t.Run("fails when node identity key (secp256k1) signature does not match", func(t *testing.T) {
		f := newTestFixture(t)
		f.k.SetControllerConnectionID(f.ctx, "connection-0")
		f.k.SetPolicyId(f.ctx, "policy-abc")
		f.ica.icaAddress = testICAAddr()
		f.ica.icaFound = true

		// A minimal but syntactically valid DER byte string that does NOT
		// correspond to the correct key / message.
		badNodeSig := []byte{0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01}

		_, _, err := f.k.RegisterEntity(
			f.ctx,
			peerPub, peerSig,
			nodePub, badNodeSig,
			message,
			types.RoleIndexer,
			signerAddr,
		)
		require.Error(t, err)
	})
}

// ─── RegisterEntity – Happy Path & Role Persistence ──────────────────────────

// Test_RegisterEntity_ValidKeys covers the successful registration flow for
// both the indexer and host roles, idempotent re-registration, and the
// duplicate-DID / duplicate-address rejection guards.
func Test_RegisterEntity_ValidKeys(t *testing.T) {
	const registrationMessage = "register-entity-message"
	message := []byte(registrationMessage)

	// genKeys generates a fresh ed25519 peer key-pair and a fresh secp256k1
	// node-identity key-pair.  Both keys produce valid signatures over message.
	genKeys := func(t *testing.T) (peerPub, peerSig, nodePub, nodeSig []byte) {
		t.Helper()

		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		peerPub = []byte(pub)
		peerSig = ed25519.Sign(priv, message)

		nodePriv, err := secp256k1.GeneratePrivateKey()
		require.NoError(t, err)
		nodePub = nodePriv.PubKey().SerializeCompressed()
		h := sha256.Sum256(message)
		nodeSig = ecdsa.Sign(nodePriv, h[:]).Serialize()
		return
	}

	// setupReady configures a fixture so that the keeper has all the
	// prerequisites fulfilled for RegisterEntity to reach the ACP layer.
	setupReady := func(t *testing.T) *testFixture {
		t.Helper()
		f := newTestFixture(t)
		f.k.SetControllerConnectionID(f.ctx, "connection-0")
		f.k.SetPolicyId(f.ctx, "policy-001")
		f.ica.icaAddress = testICAAddr()
		f.ica.icaFound = true
		return f
	}

	t.Run("successfully registers an indexer and persists addr→DID mapping", func(t *testing.T) {
		f := setupReady(t)
		peerPub, peerSig, nodePub, nodeSig := genKeys(t)
		signer := make([]byte, 20)
		signer[0] = 0x01

		did, pid, err := f.k.RegisterEntity(
			f.ctx,
			peerPub, peerSig,
			nodePub, nodeSig,
			message,
			types.RoleIndexer,
			signer,
		)
		require.NoError(t, err)
		require.NotEmpty(t, did, "DID must be non-empty")
		require.NotEmpty(t, pid, "PID must be non-empty")

		// The addr→DID mapping must be retrievable from the store.
		storedDID, found := f.k.GetDidForAddressRole(f.ctx, signer, types.RoleIndexer)
		require.True(t, found)
		require.Equal(t, did, storedDID)
	})

	t.Run("successfully registers a host and persists addr→DID mapping", func(t *testing.T) {
		f := setupReady(t)
		peerPub, peerSig, nodePub, nodeSig := genKeys(t)
		signer := make([]byte, 20)
		signer[0] = 0x02

		did, pid, err := f.k.RegisterEntity(
			f.ctx,
			peerPub, peerSig,
			nodePub, nodeSig,
			message,
			types.RoleHost,
			signer,
		)
		require.NoError(t, err)
		require.NotEmpty(t, did)
		require.NotEmpty(t, pid)

		storedDID, found := f.k.GetDidForAddressRole(f.ctx, signer, types.RoleHost)
		require.True(t, found)
		require.Equal(t, did, storedDID)
	})

	t.Run("indexer and host roles for the same address are stored independently", func(t *testing.T) {
		f := setupReady(t)
		signer := make([]byte, 20)
		signer[0] = 0x03

		// Distinct keys for each role to avoid DID collisions.
		peerPubI, peerSigI, nodePubI, nodeSigI := genKeys(t)
		peerPubH, peerSigH, nodePubH, nodeSigH := genKeys(t)

		didIndexer, _, err := f.k.RegisterEntity(
			f.ctx, peerPubI, peerSigI, nodePubI, nodeSigI, message, types.RoleIndexer, signer,
		)
		require.NoError(t, err)

		didHost, _, err := f.k.RegisterEntity(
			f.ctx, peerPubH, peerSigH, nodePubH, nodeSigH, message, types.RoleHost, signer,
		)
		require.NoError(t, err)

		require.NotEqual(t, didIndexer, didHost, "different keys must produce different DIDs")

		gotIndexerDID, foundI := f.k.GetDidForAddressRole(f.ctx, signer, types.RoleIndexer)
		require.True(t, foundI)
		require.Equal(t, didIndexer, gotIndexerDID)

		gotHostDID, foundH := f.k.GetDidForAddressRole(f.ctx, signer, types.RoleHost)
		require.True(t, foundH)
		require.Equal(t, didHost, gotHostDID)
	})

	t.Run("re-registering the same address and keys is idempotent", func(t *testing.T) {
		f := setupReady(t)
		peerPub, peerSig, nodePub, nodeSig := genKeys(t)
		signer := make([]byte, 20)
		signer[0] = 0x04

		did1, _, err := f.k.RegisterEntity(
			f.ctx, peerPub, peerSig, nodePub, nodeSig, message, types.RoleIndexer, signer,
		)
		require.NoError(t, err)

		did2, _, err := f.k.RegisterEntity(
			f.ctx, peerPub, peerSig, nodePub, nodeSig, message, types.RoleIndexer, signer,
		)
		require.NoError(t, err)
		require.Equal(t, did1, did2, "identical keys must produce the same DID both times")
	})

	t.Run("re-registering an address with different node identity key is rejected", func(t *testing.T) {
		f := setupReady(t)
		signer := make([]byte, 20)
		signer[0] = 0x05

		peerPub1, peerSig1, nodePub1, nodeSig1 := genKeys(t)
		_, _, err := f.k.RegisterEntity(
			f.ctx, peerPub1, peerSig1, nodePub1, nodeSig1, message, types.RoleIndexer, signer,
		)
		require.NoError(t, err)

		// A second key pair for the same signer address must be rejected because
		// the derived DID will differ from the one already stored.
		peerPub2, peerSig2, nodePub2, nodeSig2 := genKeys(t)
		_, _, err = f.k.RegisterEntity(
			f.ctx, peerPub2, peerSig2, nodePub2, nodeSig2, message, types.RoleIndexer, signer,
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "address already registered for this role with a different DID")
	})
}
