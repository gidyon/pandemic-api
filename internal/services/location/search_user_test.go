package location

import (
	"context"
	"github.com/gidyon/pandemic-api/pkg/api/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = Describe("Search Users #search", func() {
	var (
		searchReq *location.SearchUsersRequest
		ctx       context.Context
	)

	BeforeEach(func() {
		searchReq = &location.SearchUsersRequest{
			Query:     "Burnpaper",
			PageToken: 0,
		}
		ctx = context.Background()
	})

	Describe("Searching users with nil request", func() {
		It("should fail when request is nil", func() {
			searchReq = nil
			searchRes, err := LocationAPI.SearchUsers(context.Background(), searchReq)
			Expect(err).To(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
			Expect(searchRes).To(BeNil())
		})
	})

	When("Searching users with weird request payload", func() {
		It("should succeed when page token is weird", func() {
			searchReq.PageToken = int32(-45)
			searchRes, err := LocationAPI.SearchUsers(context.Background(), searchReq)
			Expect(err).ToNot(HaveOccurred())
			Expect(status.Code(err)).To(Equal(codes.OK))
			Expect(searchRes).ToNot(BeNil())
		})
	})

	When("Searching users with valid request", func() {
		var userName string
		Context("Lets create atleast one antmicrobial", func() {
			It("should succeed", func() {
				createReq := &location.AddUserRequest{
					User: fakeUser(),
				}
				createRes, err := LocationAPI.AddUser(ctx, createReq)
				Expect(err).ToNot(HaveOccurred())
				Expect(status.Code(err)).To(Equal(codes.OK))
				Expect(createRes).ToNot(BeNil())
				userName = createReq.User.FullName
			})
		})

		Context("Lets update search query", func() {
			BeforeEach(func() {
				searchReq.Query = userName
			})

			It("should search users", func() {
				searchRes, err := LocationAPI.SearchUsers(ctx, searchReq)
				Expect(err).ToNot(HaveOccurred())
				Expect(status.Code(err)).To(Equal(codes.OK))
				Expect(searchRes).ToNot(BeNil())
				Expect(len(searchRes.Users)).ShouldNot(BeZero())
			})

			It("should search users even when page token is large", func() {
				searchReq.PageToken = 300
				searchRes, err := LocationAPI.SearchUsers(ctx, searchReq)
				Expect(err).ToNot(HaveOccurred())
				Expect(status.Code(err)).To(Equal(codes.OK))
				Expect(searchRes).ToNot(BeNil())
				Expect(len(searchRes.Users)).Should(BeZero())
			})
		})

		Describe("Searching users with empty search query", func() {
			It("should return empty results", func() {
				searchRes, err := LocationAPI.SearchUsers(ctx, searchReq)
				Expect(err).ToNot(HaveOccurred())
				Expect(status.Code(err)).To(Equal(codes.OK))
				Expect(searchRes).ToNot(BeNil())
				Expect(len(searchRes.Users)).Should(BeZero())
			})
		})
	})
})
