package tracing

import (
	"context"
	"github.com/gidyon/pandemic-api/pkg/api/contact_tracing"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"time"

	"github.com/Pallinder/go-randomdata"
	"github.com/gidyon/pandemic-api/internal/services"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Tracing user after being infected with COVID-19 £trace", func() {
	var (
		traceReq *contact_tracing.TraceUserLocationsRequest
		ctx      context.Context
	)

	AfterEach(func() {
		traceReq = &contact_tracing.TraceUserLocationsRequest{
			PhoneNumber: randomdata.PhoneNumber(),
			SinceDate:   "2020-03-02",
		}
		ctx = context.Background()
	})

	Describe("Tracing user contact_tracings with malformed request", func() {
		It("should fail if the request is nil", func() {
			traceReq = nil
			traceRes, err := TracingAPI.TraceUserLocations(ctx, traceReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(traceRes).Should(BeNil())
		})
		It("should if phone number is missing in request", func() {
			traceReq.PhoneNumber = ""
			traceRes, err := TracingAPI.TraceUserLocations(ctx, traceReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(traceRes).Should(BeNil())
		})
		It("should if since date is missing in request", func() {
			traceReq.SinceDate = ""
			traceRes, err := TracingAPI.TraceUserLocations(ctx, traceReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(traceRes).Should(BeNil())
		})
	})

	Describe("Tracing user contact_tracings with valid request", func() {
		It("should fail is the phone is not registered", func() {
			traceRes, err := TracingAPI.TraceUserLocations(ctx, traceReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.NotFound))
			Expect(traceRes).Should(BeNil())
		})

		Describe("Tracing user contact_tracing", func() {
			var (
				patientPhone string
				operationID  int64
				userDB       *services.UserModel
				sinceDate    *time.Time
			)
			Context("Lets create the user first", func() {
				It("should create the user without error", func() {
					patientPhone = randomdata.PhoneNumber()[:10]
					userDB = &services.UserModel{
						PhoneNumber: patientPhone,
						FullName:    randomdata.FullName(randomdata.Female),
						Status:      int8(location.Status_POSITIVE),
						DeviceToken: randomdata.MacAddress(),
					}
					err := TracingServer.sqlDB.Create(userDB).Error
					Expect(err).ShouldNot(HaveOccurred())
				})
			})

			Describe("Tracing the user", func() {
				It("should succeed if the user exists", func() {
					traceReq.PhoneNumber = patientPhone
					traceRes, err := TracingAPI.TraceUserLocations(ctx, traceReq)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(status.Code(err)).Should(Equal(codes.OK))
					Expect(traceRes).ShouldNot(BeNil())
					operationID = traceRes.OperationId

					d, err := time.Parse("2006-01-02", traceReq.SinceDate)
					Expect(err).ShouldNot(HaveOccurred())
					sinceDate = &d
				})
			})

			Describe("Trace userWorker method", func() {
				It("shoould run and finish with success", func() {
					TracingServer.traceUserWorker(AlertStream, uint(operationID), []string{}, userDB, sinceDate)
				})
			})
		})
	})
})
