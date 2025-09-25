package sdk

import (
	"context"
	"fmt"
	"os"

	cmtlog "github.com/cometbft/cometbft/libs/log"
	cometrpc "github.com/cometbft/cometbft/rpc/client"
	cometrpchttp "github.com/cometbft/cometbft/rpc/client/http"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	DefaultGRPCAddr      = "localhost:9090"
	DefaultCometHTTPAddr = "http://localhost:26657"
	abciSocketPath       = "/websocket"
)

type Client struct {
	grpcAddr     string
	cometRPCAddr string
	grpcOpts     []grpc.DialOption
	logger       cmtlog.Logger

	conn        *grpc.ClientConn
	cometClient cometrpc.Client
	txClient    txtypes.ServiceClient
}

type Opt func(*Client) error

func WithGRPCAddr(addr string) Opt { return func(c *Client) error { c.grpcAddr = addr; return nil } }
func WithCometRPCAddr(addr string) Opt {
	return func(c *Client) error { c.cometRPCAddr = addr; return nil }
}
func WithGRPCOpts(opts ...grpc.DialOption) Opt {
	return func(c *Client) error { c.grpcOpts = opts; return nil }
}

func NewClient(opts ...Opt) (*Client, error) {
	c := &Client{
		grpcAddr:     DefaultGRPCAddr,
		cometRPCAddr: DefaultCometHTTPAddr,
		logger:       cmtlog.NewTMLogger(os.Stderr),
	}

	for _, o := range opts {
		if err := o(c); err != nil {
			return nil, err
		}
	}

	dial := append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, c.grpcOpts...)
	conn, err := grpc.Dial(c.grpcAddr, dial...)
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	rpc, err := cometrpchttp.New(c.cometRPCAddr, abciSocketPath)
	if err != nil {
		return nil, fmt.Errorf("comet rpc: %w", err)
	}
	if err := rpc.Start(); err != nil {
		return nil, fmt.Errorf("start comet rpc: %w", err)
	}

	c.conn = conn
	c.cometClient = rpc
	c.txClient = txtypes.NewServiceClient(conn)
	return c, nil
}

func (c *Client) Close() {
	if c.cometClient != nil {
		_ = c.cometClient.Stop()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *Client) TxClient() txtypes.ServiceClient { return c.txClient }

func (c *Client) BroadcastTx(ctx context.Context, tx sdk.Tx) (*sdk.TxResponse, error) {
	encode := authtx.DefaultTxEncoder()
	bz, err := encode(tx)
	if err != nil {
		return nil, fmt.Errorf("encode tx: %w", err)
	}

	res, err := c.txClient.BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
		TxBytes: bz,
	})
	if err != nil {
		return nil, err
	}
	if res.TxResponse.Code != 0 {
		return res.TxResponse, fmt.Errorf("rejected: %s (%d)", res.TxResponse.RawLog, res.TxResponse.Code)
	}
	return res.TxResponse, nil
}
