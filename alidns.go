package alidnsslim

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type (
	Client struct {
		accessKeyId     string
		accessKeySecret string
	}
)

// NewClient creates a client given access key ID and secret.
func NewClient(id, secret string) *Client {
	return &Client{
		accessKeyId:     id,
		accessKeySecret: secret,
	}
}

type ResponseError struct {
	Message string
	Code    string
}

func (e ResponseError) Error() string {
	return fmt.Sprintf("%s Error: %s", e.Code, e.Message)
}

// GetDomains generates params with DescribeDomains action.
func GetDomains(values ...url.Values) url.Values {
	params := url.Values{}
	params.Set("Action", "DescribeDomains")
	return merge(params, values...)
}

// GetDomainRecords generates params with DescribeDomainRecords action.
func GetDomainRecords(domainName string, values ...url.Values) url.Values {
	params := url.Values{}
	params.Set("Action", "DescribeDomainRecords")
	params.Set("DomainName", domainName)
	return merge(params, values...)
}

// GetDomainRecord generates params with DescribeDomainRecordInfo action.
func GetDomainRecord(id string, values ...url.Values) url.Values {
	params := url.Values{}
	params.Set("Action", "DescribeDomainRecordInfo")
	params.Set("RecordId", id)
	return merge(params, values...)
}

// AddDomainRecord generates params with AddDomainRecord action.
func AddDomainRecord(record, domainName, typeValue, value string, values ...url.Values) url.Values {
	params := url.Values{}
	params.Set("Action", "AddDomainRecord")
	params.Set("RR", record)
	params.Set("DomainName", domainName)
	params.Set("Type", typeValue)
	params.Set("Value", value)
	return merge(params, values...)
}

// UpdateDomainRecord generates params with UpdateDomainRecord action.
func UpdateDomainRecord(id, record, typeValue, value string, values ...url.Values) url.Values {
	params := url.Values{}
	params.Set("Action", "UpdateDomainRecord")
	params.Set("RecordId", id)
	params.Set("RR", record)
	params.Set("Type", typeValue)
	params.Set("Value", value)
	return merge(params, values...)
}

// DeleteDomainRecord generates params witht DeleteDomainRecord action.
func DeleteDomainRecord(id string, values ...url.Values) url.Values {
	params := url.Values{}
	params.Set("Action", "DeleteDomainRecord")
	params.Set("RecordId", id)
	return merge(params, values...)
}

// Page generates params with PageNumber.
func Page(page int, values ...url.Values) url.Values {
	params := url.Values{}
	params.Set("PageNumber", strconv.Itoa(page))
	return merge(params, values...)
}

// PageSize generates params with PageSize.
func PageSize(size int, values ...url.Values) url.Values {
	params := url.Values{}
	params.Set("PageSize", strconv.Itoa(size))
	return merge(params, values...)
}

// GetAll gets all paginated resources into destinations.
func (client Client) GetAll(ctx context.Context, params url.Values, dest ...interface{}) error {
	if len(dest) == 1 {
		return errors.New("dest size must not be 1")
	}
	var currentPage, totalCount, pageSize int
	var totalPages int = 1
	dest = append(dest, &currentPage, "PageNumber", &totalCount, "TotalCount", &pageSize, "PageSize")
	for currentPage < totalPages {
		if err := client.Get(ctx, merge(params, Page(currentPage+1)), dest...); err != nil {
			return err
		}
		if totalCount == 0 || pageSize == 0 {
			return nil
		}
		totalPages = int(math.Ceil(float64(totalCount) / float64(pageSize)))
	}
	return nil
}

// Do is an alias to Get. Use this to indicate action is destructive.
func (client Client) Do(ctx context.Context, params url.Values, dest ...interface{}) error {
	return client.Get(ctx, params, dest...)
}

// Get gets resources into destinations. For paginated resources, use GetAll.
func (client Client) Get(ctx context.Context, params url.Values, dest ...interface{}) error {
	ts := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params.Set("Format", "json")
	params.Set("Version", "2015-01-09")
	params.Set("AccessKeyId", client.accessKeyId)
	params.Set("SignatureMethod", "HMAC-SHA1")
	params.Set("Timestamp", ts)
	params.Set("SignatureVersion", "1.0")
	params.Set("SignatureNonce", randomString(64))
	query := buildQueryString(params)
	signature := sign(client.accessKeySecret, urlEncode(query))
	params.Set("Signature", signature)
	req, err := http.NewRequestWithContext(ctx, "GET", "https://alidns.aliyuncs.com/?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var rerr ResponseError
	json.Unmarshal(b, &rerr)
	if rerr.Code != "" {
		return rerr
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("returned status %d instead of 200", resp.StatusCode)
	}
	if len(dest) == 0 {
		return nil
	}
	if len(dest) > 1 {
		for n := 0; n < len(dest)/2; n++ {
			arrange(b, dest[2*n], dest[2*n+1].(string))
		}
		return nil
	}
	if x, ok := dest[0].(*[]byte); ok {
		*x = b
		return nil
	}
	return json.Unmarshal(b, dest[0])
}

func merge(params url.Values, values ...url.Values) url.Values {
	for _, value := range values {
		for key := range value {
			params.Set(key, value.Get(key))
		}
	}
	return params
}

func sign(secret string, query string) string {
	mac := hmac.New(sha1.New, []byte(secret+"&"))
	mac.Write([]byte("GET&%2F&" + query))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func urlEncode(input string) string {
	return strings.Replace(url.QueryEscape(input), "+", "%20", -1)
}

func buildQueryString(params url.Values) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key == "Signature" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	queries := make([]string, 0, len(params))
	for _, key := range keys {
		query := fmt.Sprintf("%s=%s", urlEncode(key), urlEncode(params.Get(key)))
		queries = append(queries, query)
	}
	queryString := strings.Join(queries, "&")
	return queryString
}

func randomString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func arrange(data []byte, target interface{}, key string) {
	keys := strings.Split(key, ".")
	baseType := reflect.TypeOf(target).Elem()
	if baseType.Kind() == reflect.Slice {
		baseType = baseType.Elem()
	}
	typ := baseType
	for i := len(keys) - 1; i > -1; i-- {
		key := keys[i]
		if key == "*" {
			typ = reflect.SliceOf(typ)
		} else if key != "" {
			typ = reflect.MapOf(reflect.TypeOf(key), typ)
		}
	}
	d := reflect.New(typ)
	json.Unmarshal(data, d.Interface())
	items := collect(d.Elem(), keys)
	v := reflect.Indirect(reflect.ValueOf(target))
	for n := range items {
		item := items[n]
		if !item.IsValid() {
			item = reflect.New(baseType).Elem()
		}
		if v.Kind() == reflect.Slice {
			v.Set(reflect.Append(v, item))
		} else {
			v.Set(item)
		}
	}
}

func collect(x reflect.Value, keys []string) (out []reflect.Value) {
	for i, key := range keys {
		if key == "*" {
			k := keys[i+1:]
			for i := 0; i < x.Len(); i++ {
				out = append(out, collect(x.Index(i), k)...)
			}
			return
		} else if key != "" {
			x = x.MapIndex(reflect.ValueOf(key))
		}
	}
	out = append(out, x)
	return
}
