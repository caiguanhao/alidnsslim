package alidnsslim

import (
	"context"
	"os"
	"testing"
)

const format = "  %-20s  %-30s  %-5s  %s"

type domainRecord struct {
	Id         string `json:"RecordId"`
	DomainName string
	RR         string
	Type       string
	Value      string
}

func TestSign(t *testing.T) {
	keyId, keySecret := os.Getenv("ALIDNS_KEY_ID"), os.Getenv("ALIDNS_KEY_SECRET")
	if keyId == "" || keySecret == "" {
		t.Fatal("Please set ALIDNS_KEY_ID and ALIDNS_KEY_SECRET environment variables.")
	}

	client := NewClient(keyId, keySecret)

	ctx := context.Background()

	var domains []string
	if err := client.GetAll(ctx, GetDomains(PageSize(2)), &domains, "Domains.Domain.*.DomainName"); err != nil {
		t.Fatal(err)
	}

	if len(domains) == 0 {
		t.Log("No domains for test")
		return
	}

	targetDomain := domains[0]

	var recordId string
	if err := client.Do(ctx, AddDomainRecord("hello", targetDomain, "TXT", "world"), &recordId, "RecordId"); err != nil {
		t.Fatal(err)
	} else {
		t.Log("Created record with id", recordId)
	}

	targetValue := "earth"

	var record domainRecord
	if err := client.Get(ctx, GetDomainRecord(recordId), &record, ""); err != nil {
		t.Fatal(err)
	} else {
		t.Logf(format, "ID", "NAME", "TYPE", "VALUE")
		t.Logf(format, record.Id, record.RR+"."+record.DomainName, record.Type, record.Value)
		if err := client.Do(ctx, UpdateDomainRecord(record.Id, record.RR, record.Type, targetValue)); err != nil {
			t.Fatal(err)
		} else {
			t.Log("Updated record with id", recordId)
		}
	}

	for _, domain := range domains {
		var records []domainRecord
		if err := client.GetAll(ctx, GetDomainRecords(domain, PageSize(2)), &records, "DomainRecords.Record.*"); err != nil {
			t.Fatal(err)
			continue
		}
		t.Log(domain, "has", len(records), "records")
		if len(records) == 0 {
			continue
		}
		t.Logf(format, "ID", "NAME", "TYPE", "VALUE")
		for _, r := range records {
			t.Logf(format, r.Id, r.RR+"."+r.DomainName, r.Type, r.Value)
			if r.DomainName == targetDomain && r.Id == record.Id {
				if r.Value == targetValue {
					t.Log("Domain record value test passed")
				} else {
					t.Errorf("Domain record value should be %s instead of %s", targetValue, r.Value)
				}
			}
		}
	}

	if recordId != "" {
		if err := client.Do(ctx, DeleteDomainRecord(recordId)); err != nil {
			t.Fatal(err)
		} else {
			t.Log("Deleted record with id", recordId)
		}
	}
}
