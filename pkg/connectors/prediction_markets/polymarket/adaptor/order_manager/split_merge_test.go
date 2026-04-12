package order_manager_test

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/GoPolymarket/polymarket-go-sdk/pkg/auth"
	"github.com/GoPolymarket/polymarket-go-sdk/pkg/ctf"
	prediction "github.com/wisp-trading/sdk/pkg/markets/prediction/types/connector"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/order_manager"
)

// ---- minimal CTF stub ----

type stubCTFClient struct {
	splitReq  *ctf.SplitPositionRequest
	splitResp ctf.SplitPositionResponse
	splitErr  error

	mergeReq  *ctf.MergePositionsRequest
	mergeResp ctf.MergePositionsResponse
	mergeErr  error
}

func (s *stubCTFClient) SplitPosition(_ context.Context, req *ctf.SplitPositionRequest) (ctf.SplitPositionResponse, error) {
	s.splitReq = req
	return s.splitResp, s.splitErr
}

func (s *stubCTFClient) MergePositions(_ context.Context, req *ctf.MergePositionsRequest) (ctf.MergePositionsResponse, error) {
	s.mergeReq = req
	return s.mergeResp, s.mergeErr
}

func (s *stubCTFClient) PrepareCondition(_ context.Context, _ *ctf.PrepareConditionRequest) (ctf.PrepareConditionResponse, error) {
	return ctf.PrepareConditionResponse{}, nil
}
func (s *stubCTFClient) ConditionID(_ context.Context, _ *ctf.ConditionIDRequest) (ctf.ConditionIDResponse, error) {
	return ctf.ConditionIDResponse{}, nil
}
func (s *stubCTFClient) CollectionID(_ context.Context, _ *ctf.CollectionIDRequest) (ctf.CollectionIDResponse, error) {
	return ctf.CollectionIDResponse{}, nil
}
func (s *stubCTFClient) PositionID(_ context.Context, _ *ctf.PositionIDRequest) (ctf.PositionIDResponse, error) {
	return ctf.PositionIDResponse{}, nil
}
func (s *stubCTFClient) RedeemPositions(_ context.Context, _ *ctf.RedeemPositionsRequest) (ctf.RedeemPositionsResponse, error) {
	return ctf.RedeemPositionsResponse{}, nil
}
func (s *stubCTFClient) RedeemNegRisk(_ context.Context, _ *ctf.RedeemNegRiskRequest) (ctf.RedeemNegRiskResponse, error) {
	return ctf.RedeemNegRiskResponse{}, nil
}

func (s *stubCTFClient) SplitPositionAsync(_ context.Context, req *ctf.SplitPositionRequest) (common.Hash, <-chan error, error) {
	s.splitReq = req
	ch := make(chan error, 1)
	ch <- s.splitErr
	return common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"), ch, s.splitErr
}

func (s *stubCTFClient) EnsureCollateralApproved(_ context.Context, _ common.Address, _ *big.Int) error {
	return nil
}

func (s *stubCTFClient) CollateralBalance(_ context.Context, _ common.Address) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (s *stubCTFClient) EnsureERC20Approved(_ context.Context, _, _ common.Address) error {
	return nil
}

func (s *stubCTFClient) EnsureConditionalApproved(_ context.Context, _ common.Address) error {
	return nil
}

// ---- helpers ----

// testMarket builds a prediction.Market with a known condition ID.
func testMarket(conditionIDHex string) prediction.Market {
	return prediction.Market{
		MarketID: prediction.MarketID(conditionIDHex),
	}
}

// usdcAddress is the native USDC on Polygon mainnet, matching usdcAddressHex in split.go.
var usdcAddress = common.HexToAddress("0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359")

// ---- specs ----

var _ = Describe("SplitPosition", func() {
	const conditionIDHex = "0xaabbccdd00000000000000000000000000000000000000000000000000000000"

	var (
		stub   *stubCTFClient
		om     order_manager.OrderManager
		market prediction.Market
		ctx    context.Context
	)

	BeforeEach(func() {
		stub = &stubCTFClient{}
		om = order_manager.NewOrderManager(nil, stub, nil, "", auth.SignatureEOA, common.Address{})
		market = testMarket(conditionIDHex)
		ctx = context.Background()
	})

	Context("when the CTF client succeeds", func() {
		BeforeEach(func() {
			stub.splitResp = ctf.SplitPositionResponse{
				TransactionHash: common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
				BlockNumber:     42,
			}
		})

		It("returns the transaction hash hex string", func() {
			txHash, _, err := om.SplitPosition(ctx, market, big.NewInt(5_000_000))
			Expect(err).ToNot(HaveOccurred())
			Expect(txHash).To(Equal("0x1111111111111111111111111111111111111111111111111111111111111111"))
		})

		It("passes the correct USDC collateral token address", func() {
			_, _, _ = om.SplitPosition(ctx, market, big.NewInt(1_000_000))
			Expect(stub.splitReq).ToNot(BeNil())
			Expect(stub.splitReq.CollateralToken).To(Equal(usdcAddress))
		})

		It("maps the market ID to the condition ID", func() {
			_, _, _ = om.SplitPosition(ctx, market, big.NewInt(1_000_000))
			Expect(stub.splitReq).ToNot(BeNil())
			Expect(stub.splitReq.ConditionID).To(Equal(common.HexToHash(conditionIDHex)))
		})

		It("uses the binary partition [1, 2]", func() {
			_, _, _ = om.SplitPosition(ctx, market, big.NewInt(1_000_000))
			Expect(stub.splitReq).ToNot(BeNil())
			Expect(stub.splitReq.Partition).To(HaveLen(2))
			Expect(stub.splitReq.Partition[0]).To(Equal(big.NewInt(1)))
			Expect(stub.splitReq.Partition[1]).To(Equal(big.NewInt(2)))
		})

		It("forwards the requested USDC amount unchanged", func() {
			amount := big.NewInt(10_000_000) // $10.00
			_, _, _ = om.SplitPosition(ctx, market, amount)
			Expect(stub.splitReq).ToNot(BeNil())
			Expect(stub.splitReq.Amount.Cmp(amount)).To(Equal(0))
		})

		It("uses an empty parent collection ID", func() {
			_, _, _ = om.SplitPosition(ctx, market, big.NewInt(1_000_000))
			Expect(stub.splitReq).ToNot(BeNil())
			Expect(stub.splitReq.ParentCollectionID).To(Equal(common.Hash{}))
		})
	})

	Context("when the CTF client returns an error", func() {
		BeforeEach(func() {
			stub.splitErr = errors.New("rpc: connection refused")
		})

		It("propagates the error", func() {
			_, _, err := om.SplitPosition(ctx, market, big.NewInt(1_000_000))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("rpc: connection refused"))
		})

		It("returns an empty transaction hash", func() {
			txHash, _, _ := om.SplitPosition(ctx, market, big.NewInt(1_000_000))
			Expect(txHash).To(BeEmpty())
		})
	})
})

var _ = Describe("MergePositions", func() {
	const conditionIDHex = "0xdeadbeef00000000000000000000000000000000000000000000000000000000"

	var (
		stub   *stubCTFClient
		om     order_manager.OrderManager
		market prediction.Market
		ctx    context.Context
	)

	BeforeEach(func() {
		stub = &stubCTFClient{}
		om = order_manager.NewOrderManager(nil, stub, nil, "", auth.SignatureEOA, common.Address{})
		market = testMarket(conditionIDHex)
		ctx = context.Background()
	})

	Context("when the CTF client succeeds", func() {
		BeforeEach(func() {
			stub.mergeResp = ctf.MergePositionsResponse{
				TransactionHash: common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
				BlockNumber:     99,
			}
		})

		It("returns the transaction hash hex string", func() {
			txHash, err := om.MergePositions(ctx, market, big.NewInt(3_000_000))
			Expect(err).ToNot(HaveOccurred())
			Expect(txHash).To(Equal("0x2222222222222222222222222222222222222222222222222222222222222222"))
		})

		It("passes the correct USDC collateral token address", func() {
			_, _ = om.MergePositions(ctx, market, big.NewInt(1_000_000))
			Expect(stub.mergeReq).ToNot(BeNil())
			Expect(stub.mergeReq.CollateralToken).To(Equal(usdcAddress))
		})

		It("maps the market ID to the condition ID", func() {
			_, _ = om.MergePositions(ctx, market, big.NewInt(1_000_000))
			Expect(stub.mergeReq).ToNot(BeNil())
			Expect(stub.mergeReq.ConditionID).To(Equal(common.HexToHash(conditionIDHex)))
		})

		It("uses the binary partition [1, 2]", func() {
			_, _ = om.MergePositions(ctx, market, big.NewInt(1_000_000))
			Expect(stub.mergeReq).ToNot(BeNil())
			Expect(stub.mergeReq.Partition).To(HaveLen(2))
			Expect(stub.mergeReq.Partition[0]).To(Equal(big.NewInt(1)))
			Expect(stub.mergeReq.Partition[1]).To(Equal(big.NewInt(2)))
		})

		It("forwards the requested USDC amount unchanged", func() {
			amount := big.NewInt(7_500_000) // $7.50
			_, _ = om.MergePositions(ctx, market, amount)
			Expect(stub.mergeReq).ToNot(BeNil())
			Expect(stub.mergeReq.Amount.Cmp(amount)).To(Equal(0))
		})

		It("uses an empty parent collection ID", func() {
			_, _ = om.MergePositions(ctx, market, big.NewInt(1_000_000))
			Expect(stub.mergeReq).ToNot(BeNil())
			Expect(stub.mergeReq.ParentCollectionID).To(Equal(common.Hash{}))
		})
	})

	Context("when the CTF client returns an error", func() {
		BeforeEach(func() {
			stub.mergeErr = errors.New("insufficient token balance")
		})

		It("propagates the error", func() {
			_, err := om.MergePositions(ctx, market, big.NewInt(1_000_000))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("insufficient token balance"))
		})

		It("returns an empty transaction hash", func() {
			txHash, _ := om.MergePositions(ctx, market, big.NewInt(1_000_000))
			Expect(txHash).To(BeEmpty())
		})
	})
})
