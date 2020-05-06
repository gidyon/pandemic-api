package location

import (
	"context"
	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Sending locations to the server #sends", func() {
	var (
		sendReq *location.SendLocationsRequest
		ctx     context.Context
	)

	BeforeEach(func() {
		sendReq = &location.SendLocationsRequest{
			UserId:   randomdata.PhoneNumber(),
			StatusId: location.Status_RECOVERED,
			Locations: []*location.Location{
				fakeLocation(), fakeLocation(), fakeLocation(), fakeLocation(), fakeLocation(), fakeLocation(), fakeLocation(),
			},
		}
		ctx = context.Background()
	})

	Describe("Adding locations with malformed request", func() {
		It("should fail if the location is nil", func() {
			sendReq = nil
			sendres, err := LocationAPI.SendLocations(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
		It("should fail when user is missing", func() {
			sendReq.UserId = ""
			sendres, err := LocationAPI.SendLocations(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
	})

	Describe("Sending locations with well-formed request", func() {
		It("should succed", func() {
			sendres, err := LocationAPI.SendLocations(ctx, sendReq)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.OK))
			Expect(sendres).ShouldNot(BeNil())
		})
	})
})
