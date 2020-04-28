package messaging

import (
	"context"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Getting messages #get", func() {
	var (
		getReq *location.GetMessagesRequest
		ctx    context.Context
	)

	BeforeEach(func() {
		getReq = &location.GetMessagesRequest{
			PhoneNumber: randomPhone(),
		}
		ctx = context.Background()
	})

	Describe("Getting messages with malformed request", func() {
		It("should fail when the request is nil", func() {
			getReq = nil
			getRes, err := MessagingAPI.GetMessages(ctx, getReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(getRes).Should(BeNil())
		})
		It("should fail when phone number is missing in request", func() {
			getReq.PhoneNumber = ""
			getRes, err := MessagingAPI.GetMessages(ctx, getReq)
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
				getReq.PhoneNumber = phoneNumber
				getRes, err := MessagingAPI.GetMessages(ctx, getReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(getRes.Messages).ShouldNot(BeNil())
				Expect(getRes.Messages).Should(HaveLen(1))
			})
		})
	})
})
