package private

import (
	"github.com/fxpgr/go-ccex-api-client/models"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type FakeRoundTripper struct {
	message  string
	status   int
	header   map[string]string
	requests []*http.Request
}

func (rt *FakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	body := strings.NewReader(rt.message)
	rt.requests = append(rt.requests, r)
	res := &http.Response{
		StatusCode: rt.status,
		Body:       ioutil.NopCloser(body),
		Header:     make(http.Header),
	}
	for k, v := range rt.header {
		res.Header.Set(k, v)
	}
	return res, nil
}

func (rt *FakeRoundTripper) Reset() {
	rt.requests = nil
}

func TestNewClient(t *testing.T) {
	_, err := NewClient("bitflyer", "APIKEY", "SECRETKEY")
	if err != nil {
		panic(err)
	}
	_, err = NewClient("poloniex", "APIKEY", "SECRETKEY")
	if err != nil {
		panic(err)
	} /*
		_, err = NewClient("hitbtc","APIKEY","SECRETKEY")
		if err != nil {
			panic(err)
		}*/
}

func newTestPrivateClient(exchangeName string, rt http.RoundTripper) PrivateClient {
	endpoint := "http://localhost:4243"
	switch strings.ToLower(exchangeName) {
	case "bitflyer":
		return &BitflyerApi{
			BaseURL:           endpoint,
			RateCacheDuration: 30 * time.Second,
			HttpClient:        http.Client{Transport: rt},
			rateMap:           nil,
			volumeMap:         nil,
			rateLastUpdated:   time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			m:                 new(sync.Mutex),
		}
	case "poloniex":
		return &PoloniexApi{
			BaseURL:           endpoint,
			RateCacheDuration: 30 * time.Second,
			HttpClient:        http.Client{Transport: rt},
			rateMap:           nil,
			volumeMap:         nil,
			rateLastUpdated:   time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			m:                 new(sync.Mutex),
		}
	}
	return nil
}

func TestBitflyerFee(t *testing.T) {
	t.Parallel()
	json := `{
  "commission_rate": 0.001
}`
	client := newTestPrivateClient("bitflyer", &FakeRoundTripper{message: json, status: http.StatusOK})
	fee, err := client.PurchaseFeeRate()
	if err != nil {
		panic(err)
	}
	if fee != 0.001 {
		t.Errorf("BitflyerPrivateApi: Expected %v. Got %v", 0.001, fee)
	}
	fee, err = client.SellFeeRate()
	if err != nil {
		panic(err)
	}
	if fee != 0.001 {
		t.Errorf("BitflyerPrivateApi: Expected %v. Got %v", 0.001, fee)
	}
	_, err = client.TransferFee()
	if err != nil {
		panic(err)
	}
}

func TestBitflyerBalances(t *testing.T) {
	t.Parallel()
	json := `[
  {
    "currency_code": "JPY",
    "amount": 1024078,
    "available": 508000
  },
  {
    "currency_code": "BTC",
    "amount": 10.24,
    "available": 4.12
  },
  {
    "currency_code": "ETH",
    "amount": 20.48,
    "available": 16.38
  }
]`
	client := newTestPrivateClient("bitflyer", &FakeRoundTripper{message: json, status: http.StatusOK})
	balances, err := client.Balances()
	if err != nil {
		panic(err)
	}
	if balances["BTC"] != 4.12 {
		t.Errorf("BitflyerPrivateApi: Expected %v. Got %v", 4.12, balances)
	}
	balanceMap, err := client.CompleteBalances()
	if err != nil {
		panic(err)
	}

	if balanceMap["BTC"].Available != 4.12 || balanceMap["BTC"].OnOrders != 6.12 {
		t.Error("BitflyerPrivateApi: balance map error")
	}
}

func TestBitflyerOrders(t *testing.T) {
	t.Parallel()
	json := `[
  {
    "id": 138398,
    "child_order_id": "JOR20150707-084555-022523",
    "product_code": "BTC_JPY",
    "side": "BUY",
    "child_order_type": "LIMIT",
    "price": 30000,
    "average_price": 30000,
    "size": 0.1,
    "child_order_state": "COMPLETED",
    "expire_date": "2015-07-14T07:25:52",
    "child_order_date": "2015-07-07T08:45:53",
    "child_order_acceptance_id": "JRF20150707-084552-031927",
    "outstanding_size": 0,
    "cancel_size": 0,
    "executed_size": 0.1,
    "total_commission": 0
  }]`
	client := newTestPrivateClient("bitflyer", &FakeRoundTripper{message: json, status: http.StatusOK})
	orders, err := client.ActiveOrders()
	if err != nil {
		panic(err)
	}
	if orders[0].Settlement != "JPY" {
		t.Errorf("BitflyerPrivateApi: Expected %v. Got %v", "JPY", orders[0].Settlement)
	}
	if orders[0].Trading != "BTC" {
		t.Errorf("BitflyerPrivateApi: Expected %v. Got %v", "BTC", orders[0].Trading)
	}
	if orders[0].ExchangeOrderID != "JRF20150707-084552-031927" {
		t.Errorf("BitflyerPrivateApi: Expected %v. Got %v", "JRF20150707-084552-031927", orders[0].ExchangeOrderID)
	}
	if orders[0].Type != models.Ask {
		t.Errorf("BitflyerPrivateApi: Expected %v. Got %v", "BUY", orders[0].Type)
	}
}

func TestBitflyerOrder(t *testing.T) {
	t.Parallel()
	json := `{
    "child_order_acceptance_id": "JRF20150707-050237-639234"
}`
	client := newTestPrivateClient("bitflyer", &FakeRoundTripper{message: json, status: http.StatusOK})
	orderId, err := client.Order("BTC", "JPY", models.Bid, 1000000, 0.01)
	if err != nil {
		panic(err)
	}
	if orderId != "JRF20150707-050237-639234" {
		t.Errorf("BitflyerPrivateApi: Expected %v. Got %v", "JRF20150707-050237-639234", orderId)
	}
	err = client.CancelOrder(orderId, "BTC_JPY")
	if err != nil {
		t.Error(err)
	}
}

func TestBitflyerOthers(t *testing.T) {
	t.Parallel()
	json := ``
	client := newTestPrivateClient("bitflyer", &FakeRoundTripper{message: json, status: http.StatusOK})
	if client.Transfer("", "", 0.1, 0.001) == nil {
		t.Errorf("transfer should not be implemented")
	}
	if _, err := client.Address(""); err == nil {
		t.Errorf("address should not be implemented")
	}
}
