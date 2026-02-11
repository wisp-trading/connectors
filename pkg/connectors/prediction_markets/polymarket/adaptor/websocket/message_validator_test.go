package websocket_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/wisp-trading/connectors/pkg/connectors/prediction_markets/polymarket/adaptor/websocket"
	"github.com/wisp-trading/connectors/pkg/websocket/security"
)

var _ = Describe("PolymarketMessageValidator", func() {
	var (
		validator security.MessageValidator
		config    websocket.ValidationConfig
	)

	BeforeEach(func() {
		config = websocket.DefaultValidationConfig()
		validator = websocket.NewMessageValidator(config)
	})

	Describe("ValidateMessage", func() {
		Context("when given valid book message", func() {
			It("should not return an error", func() {
				bookMsg := `[{
					"event_type": "book",
					"asset_id": "65818619657568813474341868652308942079804919287380422192892211131408793125422",
					"market": "0xbd31dc8a20211944f6b70f31557f1001557b59905b7738480ca09bd4532f84af",
					"bids": [{"price": "0.48", "size": "30"}],
					"asks": [{"price": "0.52", "size": "25"}],
					"timestamp": "123456789000",
					"hash": "0x0..."
				}]`

				err := validator.ValidateMessage([]byte(bookMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when given valid price_change message", func() {
			It("should not return an error", func() {
				priceChangeMsg := `[{
					"event_type": "price_change",
					"market": "0x5f65177b394277fd294cd75650044e32ba009a95022d88a0c1d565897d72f8f1",
					"price_changes": [
						{
							"asset_id": "71321045679252212594626385532706912750332728571942532289631379312455583992563",
							"price": "0.5",
							"size": "200",
							"side": "BUY",
							"hash": "56621a121a47ed9333273e21c83b660cff37ae50",
							"best_bid": "0.5",
							"best_ask": "1"
						}
					],
					"timestamp": "1757908892351"
				}]`

				err := validator.ValidateMessage([]byte(priceChangeMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when given valid tick_size_change message", func() {
			It("should not return an error", func() {
				tickSizeMsg := `[{
					"event_type": "tick_size_change",
					"asset_id": "65818619657568813474341868652308942079804919287380422192892211131408793125422",
					"market": "0xbd31dc8a20211944f6b70f31557f1001557b59905b7738480ca09bd4532f84af",
					"old_tick_size": "0.01",
					"new_tick_size": "0.001",
					"timestamp": "100000000"
				}]`

				err := validator.ValidateMessage([]byte(tickSizeMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when given valid last_trade_price message", func() {
			It("should not return an error", func() {
				lastTradeMsg := `[{
					"event_type": "last_trade_price",
					"asset_id": "114122071509644379678018727908709560226618148003371446110114509806601493071694",
					"market": "0x6a67b9d828d53862160e470329ffea5246f338ecfffdf2cab45211ec578b0347",
					"price": "0.456",
					"side": "BUY",
					"size": "219.217767",
					"timestamp": "1750428146322"
				}]`

				err := validator.ValidateMessage([]byte(lastTradeMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when given valid best_bid_ask message", func() {
			It("should not return an error", func() {
				bestBidAskMsg := `[{
					"event_type": "best_bid_ask",
					"market": "0x0005c0d312de0be897668695bae9f32b624b4a1ae8b140c49f08447fcc74f442",
					"asset_id": "85354956062430465315924116860125388538595433819574542752031640332592237464430",
					"best_bid": "0.73",
					"best_ask": "0.77",
					"spread": "0.04",
					"timestamp": "1766789469958"
				}]`

				err := validator.ValidateMessage([]byte(bestBidAskMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when given valid new_market message", func() {
			It("should not return an error", func() {
				newMarketMsg := `[{
					"event_type": "new_market",
					"id": "1031769",
					"question": "Will NVIDIA (NVDA) close above $240 end of January?",
					"market": "0x311d0c4b6671ab54af4970c06fcf58662516f5168997bdda209ec3db5aa6b0c1",
					"slug": "nvda-above-240-on-january-30-2026",
					"description": "Test description",
					"assets_ids": ["76043073756653678226373981964075571318267289248134717369284518995922789326425"],
					"outcomes": ["Yes", "No"],
					"timestamp": "1766790415550"
				}]`

				err := validator.ValidateMessage([]byte(newMarketMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when given valid market_resolved message", func() {
			It("should not return an error", func() {
				marketResolvedMsg := `[{
					"event_type": "market_resolved",
					"id": "1031769",
					"question": "Will NVIDIA (NVDA) close above $240 end of January?",
					"market": "0x311d0c4b6671ab54af4970c06fcf58662516f5168997bdda209ec3db5aa6b0c1",
					"slug": "nvda-above-240-on-january-30-2026",
					"winning_asset_id": "76043073756653678226373981964075571318267289248134717369284518995922789326425",
					"winning_outcome": "Yes",
					"timestamp": "1766790415550"
				}]`

				err := validator.ValidateMessage([]byte(marketResolvedMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when given valid multi-element array", func() {
			It("should validate all messages in the array", func() {
				multiMsg := `[
					{
						"event_type": "book",
						"asset_id": "123",
						"market": "0xabc",
						"timestamp": "1770797766421"
					},
					{
						"event_type": "last_trade_price",
						"asset_id": "456",
						"market": "0xdef",
						"timestamp": "1770797766422"
					}
				]`

				err := validator.ValidateMessage([]byte(multiMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when given empty array", func() {
			It("should return an error", func() {
				emptyMsg := `[]`

				err := validator.ValidateMessage([]byte(emptyMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("empty message array"))
			})
		})

		Context("when given nil input", func() {
			It("should return an error", func() {
				err := validator.ValidateMessage(nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid JSON array"))
			})
		})

		Context("when given empty input", func() {
			It("should return an error", func() {
				err := validator.ValidateMessage([]byte(""))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid JSON array"))
			})
		})

		Context("when given invalid JSON", func() {
			It("should return an error", func() {
				invalidMsg := `[{not valid json}]`

				err := validator.ValidateMessage([]byte(invalidMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid JSON array"))
			})
		})

		Context("when given non-array JSON", func() {
			It("should return an error", func() {
				nonArrayMsg := `{"event_type": "book", "market": "0x123"}`

				err := validator.ValidateMessage([]byte(nonArrayMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid JSON array"))
			})
		})

		Context("when message is too large", func() {
			It("should return an error", func() {
				// Create message larger than MaxMessageSize
				largeMsg := `[{"market": "` + strings.Repeat("a", config.MaxMessageSize) + `"}]`

				err := validator.ValidateMessage([]byte(largeMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("message too large"))
			})
		})

		Context("when event_type is missing", func() {
			It("should return an error", func() {
				missingTypeMsg := `[{
					"market": "0x123",
					"asset_id": "999",
					"timestamp": "1770797766421"
				}]`

				err := validator.ValidateMessage([]byte(missingTypeMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing or invalid event_type field"))
			})
		})

		Context("when event_type is empty", func() {
			It("should return an error", func() {
				emptyTypeMsg := `[{
					"event_type": "",
					"market": "0x123",
					"asset_id": "999",
					"timestamp": "1770797766421"
				}]`

				err := validator.ValidateMessage([]byte(emptyTypeMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing or invalid event_type field"))
			})
		})

		Context("when event_type is not allowed", func() {
			It("should return an error", func() {
				invalidTypeMsg := `[{
					"event_type": "invalid_type",
					"market": "0x123",
					"asset_id": "999",
					"timestamp": "1770797766421"
				}]`

				err := validator.ValidateMessage([]byte(invalidTypeMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid event_type"))
			})
		})

		Context("when required field 'market' is missing", func() {
			It("should return an error", func() {
				missingMarketMsg := `[{
					"event_type": "book",
					"asset_id": "999",
					"timestamp": "1770797766421"
				}]`

				err := validator.ValidateMessage([]byte(missingMarketMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing required field 'market'"))
			})
		})

		Context("when required field 'asset_id' is missing for book message", func() {
			It("should return an error", func() {
				missingAssetMsg := `[{
					"event_type": "book",
					"market": "0x123",
					"timestamp": "1770797766421"
				}]`

				err := validator.ValidateMessage([]byte(missingAssetMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing required field 'asset_id'"))
			})
		})

		Context("when required field 'timestamp' is missing", func() {
			It("should return an error", func() {
				missingTimestampMsg := `[{
					"event_type": "book",
					"market": "0x123",
					"asset_id": "999"
				}]`

				err := validator.ValidateMessage([]byte(missingTimestampMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("missing required field 'timestamp'"))
			})
		})

		Context("when required field has empty string value", func() {
			It("should return an error", func() {
				emptyFieldMsg := `[{
					"event_type": "book",
					"market": "",
					"asset_id": "999",
					"timestamp": "1770797766421"
				}]`

				err := validator.ValidateMessage([]byte(emptyFieldMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("empty value for required field 'market'"))
			})
		})

		Context("when required field has null value", func() {
			It("should return an error", func() {
				nullFieldMsg := `[{
					"event_type": "book",
					"market": null,
					"asset_id": "999",
					"timestamp": "1770797766421"
				}]`

				err := validator.ValidateMessage([]byte(nullFieldMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("null value for required field 'market'"))
			})
		})

		Context("when second message in array is invalid", func() {
			It("should return an error with message index", func() {
				invalidSecondMsg := `[
					{
						"event_type": "book",
						"market": "0x123",
						"asset_id": "999",
						"timestamp": "1770797766421"
					},
					{
						"event_type": "book",
						"market": "0x456",
						"asset_id": "888"
					}
				]`

				err := validator.ValidateMessage([]byte(invalidSecondMsg))
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("message[1]"))
				Expect(err.Error()).To(ContainSubstring("timestamp"))
			})
		})

		Context("edge case: message with optional fields only", func() {
			It("should validate only required fields", func() {
				minimalMsg := `[{
					"event_type": "price_change",
					"market": "0x123",
					"timestamp": "1770797766421"
				}]`

				err := validator.ValidateMessage([]byte(minimalMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("edge case: large but valid message", func() {
			It("should handle gracefully", func() {
				// Create message with many outcomes (realistic new_market scenario)
				outcomes := make([]string, 50)
				for i := 0; i < 50; i++ {
					outcomes[i] = `"Outcome` + strings.Repeat("X", 100) + `"`
				}
				largeMsg := `[{
					"event_type": "new_market",
					"market": "0x123",
					"timestamp": "1770797766421",
					"outcomes": [` + strings.Join(outcomes, ",") + `]
				}]`

				err := validator.ValidateMessage([]byte(largeMsg))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
