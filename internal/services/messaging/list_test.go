package messaging

import (
	"context"
	"github.com/gidyon/pandemic-api/pkg/api/messaging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Getting messages #list", func() {
	var (
		listReq *messaging.ListMessagesRequest
		ctx     context.Context
	)

	BeforeEach(func() {
		listReq = &messaging.ListMessagesRequest{
			PhoneNumber: randomPhone(),
		}
		ctx = context.Background()
	})

	Describe("Getting messages with malformed request", func() {
		It("should fail when the request is nil", func() {
			listReq = nil
			getRes, err := MessagingAPI.ListMessages(ctx, listReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(getRes).Should(BeNil())
		})
		It("should fail when phone number is missing in request", func() {
			listReq.PhoneNumber = ""
			getRes, err := MessagingAPI.ListMessages(ctx, listReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(getRes).Should(BeNil())
		})
	})

	Describe("Getting messages with correct request", func() {
		var phoneNumber string
		Context("Lets create a message first", func() {
			It("should succed in creating a message", func() {
				messagePB := fakeMessage()
				messageDB, err := getMessageDB(messagePB)
				Expect(err).ShouldNot(HaveOccurred())
				err = MessagingServer.sqlDB.Create(messageDB).Error
				Expect(err).ShouldNot(HaveOccurred())
				phoneNumber = messagePB.UserPhone
			})
		})

		When("Getting messages it should succeed", func() {
			It("should succeed", func() {
				listReq.PhoneNumber = phoneNumber
				getRes, err := MessagingAPI.ListMessages(ctx, listReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(getRes.Messages).ShouldNot(BeNil())
				Expect(getRes.Messages).Should(HaveLen(1))
			})
		})
	})
})
