package messaging

import (
	"context"
	"time"

	services "github.com/gidyon/pandemic-api/internal/services"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Pallinder/go-randomdata"

	"github.com/gidyon/pandemic-api/pkg/api/location"
)

var _ = Describe("Sending messages £sending", func() {
	var (
		sendReq *location.SendMessageRequest
		ctx     context.Context
	)

	BeforeEach(func() {
		sendReq = &location.SendMessageRequest{
			UserPhone: randomdata.PhoneNumber(),
			Title:     randomdata.Paragraph()[:10],
			Message:   randomdata.Paragraph(),
			Payload: map[string]string{
				"time": time.Now().String(),
				"from": randomdata.Email(),
			},
		}
		ctx = context.Background()
	})

	Describe("Sending message with malformed request", func() {
		It("should fail if request is nil", func() {
			sendReq = nil
			sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
		It("should fail if user phone is missing", func() {
			sendReq.UserPhone = ""
			sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
		It("should fail if title is missing", func() {
			sendReq.Title = ""
			sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
		It("should fail if message is missing", func() {
			sendReq.Message = ""
			sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
		It("should fail if payload is missing", func() {
			sendReq.Payload = nil
			sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
	})

	Describe("Sending message with well-formed request", func() {
		var userPhone, messageID string

		It("should fail if user is not available", func() {
			sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.NotFound))
			Expect(sendRes).Should(BeNil())
		})

		Describe("Lets create user first as they must exist", func() {
			It("should create the user without error", func() {
				userPhone = randomdata.PhoneNumber()
				err := MessagingServer.sqlDB.Create(&services.UserModel{
					PhoneNumber: userPhone,
					FullName:    randomdata.FullName(randomdata.Female),
					Status:      int8(location.Status_POSITIVE),
					DeviceToken: randomdata.MacAddress(),
				}).Error
				Expect(err).ShouldNot(HaveOccurred())
			})

			Describe("sending the user a message", func() {

				It("should succeed", func() {
					sendReq.UserPhone = userPhone
					sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(status.Code(err)).Should(Equal(codes.OK))
					Expect(sendRes).ShouldNot(BeNil())
					messageID = sendRes.MessageId
				})

				Describe("The message should be sent and saved in table", func() {
					It("should available in table", func() {
						msg := &services.GeneralMessageData{}
						err := MessagingServer.sqlDB.Table(services.GeneralMessagesTable).
							First(msg, "message_id=? AND user_phone=?", messageID, userPhone).Error
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
			})
		})
	})
})
