package tracing

import (
	"context"
	"fmt"
	"github.com/gidyon/pandemic-api/pkg/api/contact_tracing"
	"github.com/gidyon/pandemic-api/pkg/api/messaging"
	"github.com/golang/protobuf/ptypes/empty"
	"testing"

	"github.com/go-redis/redis"

	"github.com/gidyon/micros"
	"github.com/gidyon/pandemic-api/internal/services/tracing/mocks"
	"github.com/jinzhu/gorm"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func TestTracing(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tracing Suite")
}

var (
	TracingServer *tracingAPIServer
	TracingAPI    contact_tracing.ContactTracingServer
	AlertStream   *mocks.AlertContactsStream
)

// 192.168.100.10
const (
	redisAddress = "localhost:6379"
	dbAddress    = "localhost:3306"
	schema       = "fightcovid19"
)

func startDB() (*gorm.DB, error) {
	param := "charset=utf8&parseTime=true"
	dsn := fmt.Sprintf("root:hakty11@tcp(%s)/%s?%s", dbAddress, schema, param)
	return gorm.Open("mysql", dsn)
}

var _ = BeforeSuite(func() {
	ctx := context.Background()

	// Start real database
	db, err := startDB()
	handleError(err)

	redisDB := redis.NewClient(&redis.Options{
		Addr: redisAddress,
	})

	// Mock for alert contacts stream
	AlertStream = &mocks.AlertContactsStream{}
	AlertStream.On("Send", mock.Anything, mock.Anything).
		Return(nil)
	AlertStream.On("CloseAndRecv").Return(&empty.Empty{}, nil)
	AlertStream.On("CloseSend").Return(&empty.Empty{}, nil)

	// Create mock for messaging server
	messagingClient := &mocks.MessagingClientMock{}
	messagingClient.On("AlertContacts", mock.Anything, mock.Anything).
		Return(AlertStream, nil)
	messagingClient.On("BroadCastMessage", mock.Anything, mock.Anything).
		Return(&messaging.BroadCastMessageResponse{}, nil)
	messagingClient.On("SendMessage", mock.Anything, mock.Anything).
		Return(&messaging.SendMessageResponse{}, nil)
	messagingClient.On("Send", mock.Anything, mock.Anything).
		Return(nil)

	opt := &Options{
		SQLDB:           db,
		RedisClient:     redisDB,
		MessagingClient: messagingClient,
		Logger:          micros.NewLogger("messaging"),
	}

	// Create messaging server
	TracingAPI, err = NewContactTracingAPI(ctx, opt)
	Expect(err).ShouldNot(HaveOccurred())

	var ok bool
	TracingServer, ok = TracingAPI.(*tracingAPIServer)
	Expect(ok).Should(BeTrue())

	// Pasing incorrect payload
	opt.RedisClient = nil
	_, err = NewContactTracingAPI(ctx, opt)
	Expect(err).Should(HaveOccurred())

	opt.RedisClient = nil
	opt.SQLDB = nil
	_, err = NewContactTracingAPI(ctx, opt)
	Expect(err).Should(HaveOccurred())

	opt.SQLDB = db
	opt.MessagingClient = nil
	_, err = NewContactTracingAPI(ctx, opt)
	Expect(err).Should(HaveOccurred())

	opt.MessagingClient = nil
	opt.Logger = nil
	_, err = NewContactTracingAPI(ctx, opt)
	Expect(err).Should(HaveOccurred())
})

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

// Declarations for Ginkgo DSL
type Done ginkgo.Done
type Benchmarker ginkgo.Benchmarker

var GinkgoWriter = ginkgo.GinkgoWriter
var GinkgoRandomSeed = ginkgo.GinkgoRandomSeed
var GinkgoParallelNode = ginkgo.GinkgoParallelNode
var GinkgoT = ginkgo.GinkgoT
var CurrentGinkgoTestDescription = ginkgo.CurrentGinkgoTestDescription
var RunSpecs = ginkgo.RunSpecs
var RunSpecsWithDefaultAndCustomReporters = ginkgo.RunSpecsWithDefaultAndCustomReporters
var RunSpecsWithCustomReporters = ginkgo.RunSpecsWithCustomReporters
var Skip = ginkgo.Skip
var Fail = ginkgo.Fail
var GinkgoRecover = ginkgo.GinkgoRecover
var Describe = ginkgo.Describe
var FDescribe = ginkgo.FDescribe
var PDescribe = ginkgo.PDescribe
var XDescribe = ginkgo.XDescribe
var Context = ginkgo.Context
var FContext = ginkgo.FContext
var PContext = ginkgo.PContext
var XContext = ginkgo.XContext
var When = ginkgo.When
var FWhen = ginkgo.FWhen
var PWhen = ginkgo.PWhen
var XWhen = ginkgo.XWhen
var It = ginkgo.It
var FIt = ginkgo.FIt
var PIt = ginkgo.PIt
var XIt = ginkgo.XIt
var Specify = ginkgo.Specify
var FSpecify = ginkgo.FSpecify
var PSpecify = ginkgo.PSpecify
var XSpecify = ginkgo.XSpecify
var By = ginkgo.By
var Measure = ginkgo.Measure
var FMeasure = ginkgo.FMeasure
var PMeasure = ginkgo.PMeasure
var XMeasure = ginkgo.XMeasure
var BeforeSuite = ginkgo.BeforeSuite
var AfterSuite = ginkgo.AfterSuite
var SynchronizedBeforeSuite = ginkgo.SynchronizedBeforeSuite
var SynchronizedAfterSuite = ginkgo.SynchronizedAfterSuite
var BeforeEach = ginkgo.BeforeEach
var JustBeforeEach = ginkgo.JustBeforeEach
var JustAfterEach = ginkgo.JustAfterEach
var AfterEach = ginkgo.AfterEach

// Declarations for Gomega DSL
var RegisterFailHandler = gomega.RegisterFailHandler
var RegisterFailHandlerWithT = gomega.RegisterFailHandlerWithT
var RegisterTestingT = gomega.RegisterTestingT
var InterceptGomegaFailures = gomega.InterceptGomegaFailures
var Ω = gomega.Ω
var Expect = gomega.Expect
var ExpectWithOffset = gomega.ExpectWithOffset
var Eventually = gomega.Eventually
var EventuallyWithOffset = gomega.EventuallyWithOffset
var Consistently = gomega.Consistently
var ConsistentlyWithOffset = gomega.ConsistentlyWithOffset
var SetDefaultEventuallyTimeout = gomega.SetDefaultEventuallyTimeout
var SetDefaultEventuallyPollingInterval = gomega.SetDefaultEventuallyPollingInterval
var SetDefaultConsistentlyDuration = gomega.SetDefaultConsistentlyDuration
var SetDefaultConsistentlyPollingInterval = gomega.SetDefaultConsistentlyPollingInterval
var NewWithT = gomega.NewWithT
var NewGomegaWithT = gomega.NewGomegaWithT

// Declarations for Gomega Matchers
var Equal = gomega.Equal
var BeEquivalentTo = gomega.BeEquivalentTo
var BeIdenticalTo = gomega.BeIdenticalTo
var BeNil = gomega.BeNil
var BeTrue = gomega.BeTrue
var BeFalse = gomega.BeFalse
var HaveOccurred = gomega.HaveOccurred
var Succeed = gomega.Succeed
var MatchError = gomega.MatchError
var BeClosed = gomega.BeClosed
var Receive = gomega.Receive
var BeSent = gomega.BeSent
var MatchRegexp = gomega.MatchRegexp
var ContainSubstring = gomega.ContainSubstring
var HavePrefix = gomega.HavePrefix
var HaveSuffix = gomega.HaveSuffix
var MatchJSON = gomega.MatchJSON
var MatchXML = gomega.MatchXML
var MatchYAML = gomega.MatchYAML
var BeEmpty = gomega.BeEmpty
var HaveLen = gomega.HaveLen
var HaveCap = gomega.HaveCap
var BeZero = gomega.BeZero
var ContainElement = gomega.ContainElement
var BeElementOf = gomega.BeElementOf
var ConsistOf = gomega.ConsistOf
var HaveKey = gomega.HaveKey
var HaveKeyWithValue = gomega.HaveKeyWithValue
var BeNumerically = gomega.BeNumerically
var BeTemporally = gomega.BeTemporally
var BeAssignableToTypeOf = gomega.BeAssignableToTypeOf
var Panic = gomega.Panic
var BeAnExistingFile = gomega.BeAnExistingFile
var BeARegularFile = gomega.BeARegularFile
var BeADirectory = gomega.BeADirectory
var And = gomega.And
var SatisfyAll = gomega.SatisfyAll
var Or = gomega.Or
var SatisfyAny = gomega.SatisfyAny
var Not = gomega.Not
var WithTransform = gomega.WithTransform
