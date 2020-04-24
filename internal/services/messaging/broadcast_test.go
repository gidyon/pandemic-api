package messaging

import (
	"context"
	"time"

	"github.com/Pallinder/go-randomdata"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/gidyon/pandemic-api/pkg/api/location"
)

var _ = Describe("Broadcasting a message to many users £broadcast", func() {
	var (
		broadCastReq *location.BroadCastMessageRequest
		ctx          context.Context
	)

	BeforeEach(func() {
		broadCastReq = &location.BroadCastMessageRequest{
			Title:   randomdata.Paragraph()[:10],
			Message: randomdata.Paragraph(),
			Filters: []location.BroadCastMessageFilter{
				location.BroadCastMessageFilter_ALL,
				// location.BroadCastMessageFilter_POSITIVES,
				// location.BroadCastMessageFilter_BY_COUNTY,
			},
			Topics: []string{randomdata.State(randomdata.Small)},
			Payload: map[string]string{
				"time":  time.Now().String(),
				"from":  randomdata.Email(),
				"topic": "iwd",
			},
		}
		ctx = context.Background()
	})

	Describe("Broadcasting message with malf-formed request", func() {
		It("should fail when the request is nil", func() {
			broadCastReq = nil
			sendRes, err := MessagingAPI.BroadCastMessage(ctx, broadCastReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
		It("should fail if title is missing", func() {
			broadCastReq.Title = ""
			sendRes, err := MessagingAPI.BroadCastMessage(ctx, broadCastReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
		It("should fail if message is missing", func() {
			broadCastReq.Message = ""
			sendRes, err := MessagingAPI.BroadCastMessage(ctx, broadCastReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
		It("should fail if payload is missing", func() {
			broadCastReq.Payload = nil
			sendRes, err := MessagingAPI.BroadCastMessage(ctx, broadCastReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
		It("should fail if topics is missing", func() {
			broadCastReq.Topics = nil
			sendRes, err := MessagingAPI.BroadCastMessage(ctx, broadCastReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendRes).Should(BeNil())
		})
	})

	Describe("Broadcasting message with a well formed request", func() {
		var broadCastID string

		It("should succeed in broadcasting user message", func() {
			broadCastRes, err := MessagingAPI.BroadCastMessage(ctx, broadCastReq)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.OK))
			Expect(broadCastRes).ShouldNot(BeNil())
			Expect(broadCastRes.BroadcastMessageId).ShouldNot(BeZero())
			broadCastID = broadCastRes.BroadcastMessageId
		})

		Describe("Testing  broadCastMessage method", func() {
			It("should execute successfully", func() {
				MessagingServer.broadCastMessage(broadCastReq, broadCastID)
			})
		})
	})
})
