package messaging

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	services "github.com/gidyon/pandemic-api/internal/services"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Pallinder/go-randomdata"

	"github.com/gidyon/pandemic-api/pkg/api/location"
)

func randomType() location.MessageType {
	return location.MessageType(rand.Intn(len(location.MessageType_name)))
}

func randomPhone() string {
	return randomdata.PhoneNumber()[:12]
}

func fakeMessage() *location.Message {
	return &location.Message{
		UserPhone:    randomPhone(),
		Title:        randomdata.Paragraph()[:10],
		Notification: randomdata.Paragraph(),
		DateTime:     time.Now().Local().String()[:15],
		Sent:         false,
		Type:         randomType(),
		Data: map[string]string{
			"time": time.Now().String(),
			"from": randomdata.Email(),
		},
	}
}

var _ = Describe("Sending messages £sending", func() {
	var (
		sendReq *location.Message
		ctx     context.Context
	)

	BeforeEach(func() {
		sendReq = fakeMessage()
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
		It("should fail if notification is missing", func() {
			sendReq.Notification = ""
			sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
		It("should fail if data is missing", func() {
			sendReq.Data = nil
			sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
	})

	Describe("Sending message with well-formed request", func() {
		var (
			userPhone string
			messageID int
		)

		It("should fail if user is not available", func() {
			sendRes, err := MessagingAPI.SendMessage(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.NotFound))
			Expect(sendRes).Should(BeNil())
		})

		Describe("Lets create user first as they must exist", func() {
			It("should create the user without error", func() {
				userPhone = randomPhone()
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
					messageID, err = strconv.Atoi(sendRes.MessageId)
					Expect(err).ShouldNot(HaveOccurred())
				})

				Describe("The message should be sent and saved in table", func() {
					It("should available in table", func() {
						msg := &services.Message{}
						err := MessagingServer.sqlDB.Table(services.MessagesTable).
							First(msg, "ID=? AND user_phone=?", messageID, userPhone).Error
						Expect(err).ShouldNot(HaveOccurred())
					})
				})
			})
		})
	})
})
