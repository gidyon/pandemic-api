package location

import (
	"context"
	"fmt"
	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

func fakeLocation() *location.Location {
	now := time.Now()
	return &location.Location{
		Longitude:     float32(randomdata.Decimal(-40, 40)),
		Latitude:      float32(randomdata.Decimal(-40, 40)),
		Timestamp:     now.Unix(),
		TimeId:        fmt.Sprint(int(((now.Hour() * 60) + now.Minute()) / 5)),
		Altitude:      float32(randomdata.Decimal(0, 200)),
		Accuracy:      float32(randomdata.Decimal(0, 1)),
		Speed:         float32(randomdata.Decimal(0, 100)),
		SpeedAccuracy: float32(randomdata.Decimal(0, 1)),
		GeoFenceId:    randomdata.MacAddress(),
		Placemark:     randomdata.State(randomdata.Large),
	}
}

var _ = Describe("Sending user location #sendloc", func() {
	var (
		sendReq *location.SendLocationRequest
		ctx     context.Context
	)

	BeforeEach(func() {
		sendReq = &location.SendLocationRequest{
			UserId:   randomdata.PhoneNumber(),
			StatusId: location.Status_POSITIVE,
			Location: fakeLocation(),
		}
		ctx = context.Background()
	})

	Describe("Sending location with malformed request", func() {
		It("should fail when the request is nil", func() {
			sendReq = nil
			sendres, err := LocationAPI.SendLocation(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
		It("should fail when user is missing", func() {
			sendReq.UserId = ""
			sendres, err := LocationAPI.SendLocation(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
		It("should fail when location is missing", func() {
			sendReq.Location = nil
			sendres, err := LocationAPI.SendLocation(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
		It("should fail when location longitude is missing", func() {
			sendReq.Location.Longitude = 0
			sendres, err := LocationAPI.SendLocation(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
		It("should fail when location latitude is missing", func() {
			sendReq.Location.Latitude = 0
			sendres, err := LocationAPI.SendLocation(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
		It("should fail when location geo fence id is missing", func() {
			sendReq.Location.GeoFenceId = ""
			sendres, err := LocationAPI.SendLocation(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
		It("should fail when location timestamp is missing", func() {
			sendReq.Location.Timestamp = 0
			sendres, err := LocationAPI.SendLocation(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
		It("should fail when location time id is missing", func() {
			sendReq.Location.TimeId = ""
			sendres, err := LocationAPI.SendLocation(ctx, sendReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(sendres).Should(BeNil())
		})
	})

	When("Sending location with well-formed request", func() {
		It("should succedd", func() {
			sendres, err := LocationAPI.SendLocation(ctx, sendReq)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.OK))
			Expect(sendres).ShouldNot(BeNil())
		})
	})
})
