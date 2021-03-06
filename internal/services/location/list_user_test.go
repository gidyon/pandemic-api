package location

import (
	"context"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Listing users #list", func() {
	var (
		listReq *location.ListUsersRequest
		ctx     context.Context
	)

	BeforeEach(func() {
		listReq = &location.ListUsersRequest{
			FilterStatus: location.Status_RECOVERED,
		}
		ctx = context.Background()
	})

	Describe("Listing users", func() {
		It("should fail when the request is nil", func() {
			listReq = nil
			listRes, err := LocationAPI.ListUsers(ctx, listReq)
			Expect(err).Should(HaveOccurred())
			Expect(status.Code(err)).Should(Equal(codes.InvalidArgument))
			Expect(listRes).Should(BeNil())
		})
	})

	Describe("Listing users with well-formed request", func() {
		Describe("Create user first", func() {
			It("should succeed", func() {
				addReq := &location.AddUserRequest{
					User: fakeUser(),
				}
				addRes, err := LocationAPI.AddUser(ctx, addReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(addRes).ShouldNot(BeNil())
			})
		})

		Describe("Listing users", func() {
			It("should succeed when status is negative", func() {
				listReq.FilterStatus = location.Status_NEGATIVE
				listRes, err := LocationAPI.ListUsers(ctx, listReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(listRes).ShouldNot(BeNil())

				// Status must be negative
				for _, userPB := range listRes.Users {
					Expect(userPB.Status).Should(Equal(location.Status_NEGATIVE))
				}
			})

			It("should succeed when status is positive", func() {
				listReq.FilterStatus = location.Status_POSITIVE
				listRes, err := LocationAPI.ListUsers(ctx, listReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(listRes).ShouldNot(BeNil())

				// Status must be positive
				for _, userPB := range listRes.Users {
					Expect(userPB.Status).Should(Equal(location.Status_POSITIVE))
				}
			})

			It("should succeed when status is recovered", func() {
				listReq.FilterStatus = location.Status_RECOVERED
				listRes, err := LocationAPI.ListUsers(ctx, listReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(listRes).ShouldNot(BeNil())

				// Status must be recovered
				for _, userPB := range listRes.Users {
					Expect(userPB.Status).Should(Equal(location.Status_RECOVERED))
				}
			})

			It("should succeed when status is suspected", func() {
				listReq.FilterStatus = location.Status_SUSPECTED
				listRes, err := LocationAPI.ListUsers(ctx, listReq)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(status.Code(err)).Should(Equal(codes.OK))
				Expect(listRes).ShouldNot(BeNil())

				// Status must be suspected
				for _, userPB := range listRes.Users {
					Expect(userPB.Status).Should(Equal(location.Status_SUSPECTED))
				}
			})
		})
	})
})
