# alidnsslim

## Usage

```go
import "github.com/caiguanhao/alidnsslim"

client := alidnsslim.NewClient(ACCESS_KEY_ID, ACCESS_KEY_SECRET)

ctx := context.Background()

// get all domains
var domains []string
client.GetAll(ctx, alidnsslim.GetDomains(), &domains, "Domains.Domain.*.DomainName")

// get all records of a domain
var records []struct {
	Id         string `json:"RecordId"`
	DomainName string
	RR         string
	Type       string
	Value      string
}
client.GetAll(ctx, alidnsslim.GetDomainRecords("example.com", alidnsslim.PageSize(100)), &records, "DomainRecords.Record.*")

// create hello.example.com TXT record
var recordId string
client.Do(ctx, AddDomainRecord("hello", "example.com", "TXT", "world"), &recordId, "RecordId")
```
