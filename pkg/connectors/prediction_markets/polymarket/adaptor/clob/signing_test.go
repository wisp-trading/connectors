package clob_test

//var _ = Describe("OrderSigner", func() {
//	var (
//		signer     *adaptor.OrderSigner
//		privateKey string
//		chainID    int
//	)
//
//	BeforeEach(func() {
//		// Test private key (do not use in production)
//		privateKey = "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
//		chainID = 137 // Polygon
//
//		var err error
//		signer, err = adaptor.NewOrderSigner(privateKey, chainID)
//		Expect(err).ToNot(HaveOccurred())
//		Expect(signer).ToNot(BeNil())
//	})
//
//	Describe("NewOrderSigner", func() {
//		Context("when given valid private key", func() {
//			It("should create a signer successfully", func() {
//				s, err := adaptor.NewOrderSigner(privateKey, chainID)
//				Expect(err).ToNot(HaveOccurred())
//				Expect(s).ToNot(BeNil())
//			})
//
//			It("should accept private key with 0x prefix", func() {
//				s, err := adaptor.NewOrderSigner("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", chainID)
//				Expect(err).ToNot(HaveOccurred())
//				Expect(s).ToNot(BeNil())
//			})
//
//			It("should accept private key without 0x prefix", func() {
//				s, err := adaptor.NewOrderSigner("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", chainID)
//				Expect(err).ToNot(HaveOccurred())
//				Expect(s).ToNot(BeNil())
//			})
//		})
//
//		Context("when given invalid private key", func() {
//			It("should return an error for empty key", func() {
//				s, err := adaptor.NewOrderSigner("", chainID)
//				Expect(err).To(HaveOccurred())
//				Expect(s).To(BeNil())
//				Expect(err.Error()).To(ContainSubstring("private key"))
//			})
//
//			It("should return an error for short key", func() {
//				s, err := adaptor.NewOrderSigner("0x123", chainID)
//				Expect(err).To(HaveOccurred())
//				Expect(s).To(BeNil())
//			})
//
//			It("should return an error for non-hex key", func() {
//				s, err := adaptor.NewOrderSigner("not-a-hex-string", chainID)
//				Expect(err).To(HaveOccurred())
//				Expect(s).To(BeNil())
//			})
//		})
//
//		Context("when given invalid chain ID", func() {
//			It("should return an error for zero chain ID", func() {
//				s, err := adaptor.NewOrderSigner(privateKey, 0)
//				Expect(err).To(HaveOccurred())
//				Expect(s).To(BeNil())
//				Expect(err.Error()).To(ContainSubstring("chain ID"))
//			})
//
//			It("should return an error for negative chain ID", func() {
//				s, err := adaptor.NewOrderSigner(privateKey, -1)
//				Expect(err).To(HaveOccurred())
//				Expect(s).To(BeNil())
//			})
//		})
//	})
//
//	Describe("SignOrder", func() {
//		var order adaptor.Order
//
//		BeforeEach(func() {
//			order = adaptor.Order{
//				Salt:          1234567890,
//				Maker:         "0x1234567890123456789012345678901234567890",
//				Signer:        "0x1234567890123456789012345678901234567890",
//				Taker:         "0x0000000000000000000000000000000000000000",
//				TokenID:       "12345",
//				MakerAmount:   "1000000",
//				TakerAmount:   "500000",
//				Side:          "BUY",
//				FeeRateBps:    "100",
//				Nonce:         "1",
//				SignatureType: 2,
//				Expiration:    time.Now().Add(24 * time.Hour).Unix(),
//			}
//		})
//
//		Context("when given valid order", func() {
//			It("should return a valid signature", func() {
//				signature, err := signer.SignOrder(order)
//				Expect(err).ToNot(HaveOccurred())
//				Expect(signature).ToNot(BeEmpty())
//				Expect(signature).To(HavePrefix("0x"))
//				// Ethereum signature is 65 bytes = 130 hex chars + "0x" prefix
//				Expect(len(signature)).To(Equal(132))
//			})
//
//			It("should produce consistent signatures for same order", func() {
//				sig1, err := signer.SignOrder(order)
//				Expect(err).ToNot(HaveOccurred())
//
//				sig2, err := signer.SignOrder(order)
//				Expect(err).ToNot(HaveOccurred())
//
//				Expect(sig1).To(Equal(sig2))
//			})
//
//			It("should produce different signatures for different orders", func() {
//				sig1, err := signer.SignOrder(order)
//				Expect(err).ToNot(HaveOccurred())
//
//				order.MakerAmount = "2000000"
//				sig2, err := signer.SignOrder(order)
//				Expect(err).ToNot(HaveOccurred())
//
//				Expect(sig1).ToNot(Equal(sig2))
//			})
//		})
//
//		Context("when given nil order", func() {
//			It("should return an error", func() {
//				signature, err := signer.SignOrder(adaptor.Order{})
//				Expect(err).To(HaveOccurred())
//				Expect(signature).To(BeEmpty())
//				Expect(err.Error()).To(ContainSubstring("order cannot be nil"))
//			})
//		})
//
//		Context("when given order with missing fields", func() {
//			It("should return an error for empty maker", func() {
//				order.Maker = ""
//				signature, err := signer.SignOrder(order)
//				Expect(err).To(HaveOccurred())
//				Expect(signature).To(BeEmpty())
//			})
//
//			It("should return an error for empty token ID", func() {
//				order.TokenID = ""
//				signature, err := signer.SignOrder(order)
//				Expect(err).To(HaveOccurred())
//				Expect(signature).To(BeEmpty())
//			})
//
//			It("should return an error for zero maker amount", func() {
//				order.MakerAmount = ""
//				signature, err := signer.SignOrder(order)
//				Expect(err).To(HaveOccurred())
//				Expect(signature).To(BeEmpty())
//			})
//		})
//
//		Context("edge case: large amounts", func() {
//			It("should handle large maker amounts", func() {
//				order.MakerAmount = "999999999999999999"
//				signature, err := signer.SignOrder(order)
//				Expect(err).ToNot(HaveOccurred())
//				Expect(signature).ToNot(BeEmpty())
//			})
//
//			It("should handle large taker amounts", func() {
//				order.TakerAmount = "999999999999999999"
//				signature, err := signer.SignOrder(order)
//				Expect(err).ToNot(HaveOccurred())
//				Expect(signature).ToNot(BeEmpty())
//			})
//		})
//	})
//
//	Describe("GetAddress", func() {
//		Context("when signer is initialized", func() {
//			It("should return the Ethereum address", func() {
//				address := signer.GetAddress()
//				Expect(address).ToNot(BeEmpty())
//				Expect(address).To(HavePrefix("0x"))
//				Expect(len(address)).To(Equal(42)) // 0x + 40 hex chars
//			})
//		})
//	})
//
//	Describe("GenerateSalt", func() {
//		Context("when called", func() {
//			It("should generate a salt value", func() {
//				salt := signer.GenerateSalt()
//				Expect(salt).To(BeNumerically(">", 0))
//			})
//
//			It("should generate different salts on repeated calls", func() {
//				salt1 := signer.GenerateSalt()
//				salt2 := signer.GenerateSalt()
//
//				Expect(salt1).To(BeNumerically(">", 0))
//				Expect(salt2).To(BeNumerically(">", 0))
//			})
//		})
//	})
//})
