package keeper

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	viewbundle "github.com/shinzonetwork/viewbundle-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/shinzonetwork/shinzohub/x/view/types"
)

type queryServer struct {
	Keeper
}

var _ types.QueryServer = queryServer{}

var metadataRootTypeRe = regexp.MustCompile(`\btype\s+([A-Za-z0-9_]+)\b`)

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

func hasMetadataFilter(req *types.QueryViewsRequest) bool {
	return req.MetadataRootType != "" ||
		req.MetadataLensHash != "" ||
		req.MetadataQueryContains != "" ||
		req.MetadataSdlContains != "" ||
		req.MetadataLensArgsContains != ""
}

func extractRootType(sdl string) string {
	matches := metadataRootTypeRe.FindStringSubmatch(sdl)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func buildShortLensHash(wasm []byte) string {
	hash := crypto.Keccak256(wasm)
	return "0x" + hex.EncodeToString(hash[:16])
}

func deriveViewMetadata(data []byte) (*types.ViewMetadata, error) {
	view, err := viewbundle.NewBundler().UnbundleView(data)
	if err != nil {
		return nil, err
	}

	metadata := &types.ViewMetadata{
		Query:    view.Query,
		Sdl:      view.Sdl,
		RootType: extractRootType(view.Sdl),
		Lenses:   make([]types.ViewLensMetadata, 0, len(view.Transform.Lenses)),
	}

	for index, lens := range view.Transform.Lenses {
		wasm, err := base64.StdEncoding.DecodeString(lens.Path)
		if err != nil {
			return nil, err
		}

		metadata.Lenses = append(metadata.Lenses, types.ViewLensMetadata{
			Id:   uint32(index + 1),
			Args: lens.Arguments,
			Hash: buildShortLensHash(wasm),
		})
	}

	return metadata, nil
}

func metadataMatches(metadata *types.ViewMetadata, req *types.QueryViewsRequest) bool {
	if req.MetadataRootType != "" && metadata.RootType != req.MetadataRootType {
		return false
	}
	if req.MetadataQueryContains != "" && !strings.Contains(metadata.Query, req.MetadataQueryContains) {
		return false
	}
	if req.MetadataSdlContains != "" && !strings.Contains(metadata.Sdl, req.MetadataSdlContains) {
		return false
	}

	if req.MetadataLensHash != "" || req.MetadataLensArgsContains != "" {
		foundHash := req.MetadataLensHash == ""
		foundArgs := req.MetadataLensArgsContains == ""

		for _, lens := range metadata.Lenses {
			if req.MetadataLensHash != "" && lens.Hash == req.MetadataLensHash {
				foundHash = true
			}
			if req.MetadataLensArgsContains != "" && strings.Contains(lens.Args, req.MetadataLensArgsContains) {
				foundArgs = true
			}
		}

		if !foundHash || !foundArgs {
			return false
		}
	}

	return true
}

func containsFold(value, needle string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(needle))
}

// Views handles the list endpoint exposed by the gRPC service and REST gateway
// at GET /shinzonetwork/view/v1/views.
//
// Params:
//   - pagination.key, pagination.offset, pagination.limit,
//     pagination.count_total, pagination.reverse: pagination applied over
//     matching views after filters.
//   - include_data: include raw viewbundle bytes in View.data.
//   - since_block: minimum View.height.
//   - name: case-insensitive substring filter over View.name.
//   - creator: exact filter over View.creator.
//   - include_metadata: attach query, sdl, root_type, lenses, and parse_error.
//   - metadata_root_type, metadata_lens_hash: exact filters over parsed
//     metadata. Lens hashes are first 16 Keccak-256 bytes of decoded WASM.
//   - metadata_query_contains, metadata_sdl_contains,
//     metadata_lens_args_contains: substring filters over parsed metadata.
//
// Metadata filters parse bundles even when include_metadata is false. Parse
// failures are excluded from metadata-filtered lists and surfaced as
// metadata.parse_error only when include_metadata is true.
func (q queryServer) Views(goCtx context.Context, req *types.QueryViewsRequest) (*types.QueryViewsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	needsMetadata := req.IncludeMetadata || hasMetadataFilter(req)
	filtered := make([]types.View, 0)
	pageRes, err := q.Keeper.FilterViews(ctx, req.Pagination, func(v types.View, accumulate bool) (bool, error) {
		if req.SinceBlock > 0 && v.Height < req.SinceBlock {
			return false, nil
		}
		if req.Name != "" && !containsFold(v.Name, req.Name) {
			return false, nil
		}
		if req.Creator != "" && v.Creator != req.Creator {
			return false, nil
		}

		if needsMetadata {
			metadata, err := deriveViewMetadata(v.Data)
			if err != nil {
				if hasMetadataFilter(req) {
					return false, nil
				}
				if req.IncludeMetadata {
					v.Metadata = &types.ViewMetadata{ParseError: err.Error()}
				}
			} else {
				if !metadataMatches(metadata, req) {
					return false, nil
				}
				if req.IncludeMetadata {
					v.Metadata = metadata
				}
			}
		}

		if !req.IncludeData {
			v.Data = nil
		}

		if accumulate {
			filtered = append(filtered, v)
		}
		return true, nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryViewsResponse{
		Views:      filtered,
		Pagination: pageRes,
	}, nil
}

// View handles the single-view endpoint exposed by the gRPC service and REST
// gateway at GET /shinzonetwork/view/v1/views/{contract_address}.
//
// Params:
//   - contract_address: path key.
//   - include_data: include raw viewbundle bytes in View.data.
//   - include_metadata: attach query, sdl, root_type, lenses, and parse_error.
//
// No metadata filters apply here; include_metadata parse failures return the
// view with metadata.parse_error populated.
func (q queryServer) View(goCtx context.Context, req *types.QueryViewRequest) (*types.QueryViewResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	view, found, err := q.Keeper.GetView(ctx, req.ContractAddress)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !found {
		return nil, status.Error(codes.NotFound, "view not found")
	}

	if req.IncludeMetadata {
		metadata, err := deriveViewMetadata(view.Data)
		if err != nil {
			view.Metadata = &types.ViewMetadata{ParseError: err.Error()}
		} else {
			view.Metadata = metadata
		}
	}

	if !req.IncludeData {
		view.Data = nil
	}

	return &types.QueryViewResponse{View: view}, nil
}

func (q queryServer) ViewCount(goCtx context.Context, req *types.QueryViewCountRequest) (*types.QueryViewCountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	count := q.Keeper.GetViewCount(ctx)

	return &types.QueryViewCountResponse{Count: count}, nil
}
