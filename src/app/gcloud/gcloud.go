// Copyright © 2023 Esko Luontola
// This software is released under the Apache License 2.0.
// The license text is at http://www.apache.org/licenses/LICENSE-2.0

package gcloud

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"

	"google.golang.org/api/dns/v1"
	"google.golang.org/api/option"
)

type Client struct {
	project    string
	context    context.Context
	dnsService *dns.Service
}

func Configure(project string) *Client {
	// Support two authentication methods:
	// 1. GOOGLE_CREDENTIALS_JSON - JSON content directly in env var
	// 2. GOOGLE_APPLICATION_CREDENTIALS - path to JSON file
	if err := setupCredentials(); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	dnsService, err := dns.NewService(ctx,
		option.WithScopes(dns.NdevClouddnsReadwriteScope))
	if err != nil {
		log.Fatal(err)
	}

	return &Client{
		project:    project,
		context:    ctx,
		dnsService: dnsService,
	}
}

// setupCredentials configures GCP authentication from environment variables.
// Supports GOOGLE_CREDENTIALS_JSON (JSON content) or GOOGLE_APPLICATION_CREDENTIALS (file path).
func setupCredentials() error {
	// Check if credentials JSON is provided directly
	credentialsJSON := os.Getenv("GOOGLE_CREDENTIALS_JSON")
	if credentialsJSON != "" {
		// Write JSON to temp file and set GOOGLE_APPLICATION_CREDENTIALS
		tmpFile, err := os.CreateTemp("", "gcp-credentials-*.json")
		if err != nil {
			return fmt.Errorf("failed to create temp credentials file: %w", err)
		}
		if _, err := tmpFile.WriteString(credentialsJSON); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to write credentials to temp file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			os.Remove(tmpFile.Name())
			return fmt.Errorf("failed to close temp credentials file: %w", err)
		}
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tmpFile.Name())
		log.Printf("Using credentials from GOOGLE_CREDENTIALS_JSON environment variable")
		return nil
	}

	// Fall back to file-based credentials
	googleApplicationCredentials := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if googleApplicationCredentials == "" {
		return fmt.Errorf("no GCP credentials configured. Set either:\n" +
			"  - GOOGLE_CREDENTIALS_JSON: JSON content of service account key\n" +
			"  - GOOGLE_APPLICATION_CREDENTIALS: path to service account JSON file\n" +
			"See https://cloud.google.com/docs/authentication/production for instructions")
	}

	log.Printf("Using credentials from file: %s", googleApplicationCredentials)
	return nil
}

func (c *Client) DnsRecordsByNameAndType(names []string, recordType string) (DnsRecords, error) {
	records, err := c.DnsRecords()
	if err != nil {
		return nil, err
	}
	found := filterDnsRecordsByName(records, names)
	found = filterDnsRecordsByType(found, recordType)
	if len(found) != len(names) {
		return nil, fmt.Errorf("expected DNS records <%v> of type <%v>, but only found <%v> of them from the available <%v>",
			strings.Join(names, ", "),
			recordType,
			strings.Join(found.NamesAndTypes(), ", "),
			strings.Join(records.NamesAndTypes(), ", "))
	}
	return found, nil
}

func filterDnsRecordsByName(records DnsRecords, names []string) DnsRecords {
	if len(names) == 0 {
		return DnsRecords{}
	}
	var results DnsRecords
	for _, record := range records {
		if equalsAny(record.Name, names) {
			results = append(results, record)
		}
	}
	return results
}

func equalsAny(haystack string, needles []string) bool {
	for _, needle := range needles {
		if haystack == needle {
			return true
		}
	}
	return false
}

func filterDnsRecordsByType(records DnsRecords, recordType string) DnsRecords {
	var results DnsRecords
	for _, record := range records {
		if record.Type == recordType {
			results = append(results, record)
		}
	}
	return results
}

func (c *Client) DnsRecords() (DnsRecords, error) {
	zones, err := c.ManagedZones()
	if err != nil {
		return nil, err
	}
	var records DnsRecords
	for _, zone := range zones {
		rrsets, err := c.ResourceRecordSets(zone.Name)
		if err != nil {
			return nil, err
		}
		for _, rrset := range rrsets {
			record := &DnsRecord{ManagedZone: zone.Name, ResourceRecordSet: rrset}
			records = append(records, record)
		}

	}
	return records, nil
}

func (c *Client) ManagedZones() ([]*dns.ManagedZone, error) {
	var results []*dns.ManagedZone
	err := c.dnsService.ManagedZones.List(c.project).Pages(c.context, func(page *dns.ManagedZonesListResponse) error {
		results = append(results, page.ManagedZones...)
		return nil
	})
	return results, err
}

func (c *Client) ResourceRecordSets(managedZone string) ([]*dns.ResourceRecordSet, error) {
	var results []*dns.ResourceRecordSet
	req := c.dnsService.ResourceRecordSets.List(c.project, managedZone)
	err := req.Pages(c.context, func(page *dns.ResourceRecordSetsListResponse) error {
		results = append(results, page.Rrsets...)
		return nil
	})
	return results, err
}

func (c *Client) UpdateDnsRecords(records DnsRecords, newValues []string) (DnsRecords, error) {
	var updated DnsRecords
	for managedZone, recordsInZone := range records.GroupByZone() {
		plannedChanges := changesToUpdateDnsRecordValues(recordsInZone, newValues)
		if plannedChanges == nil {
			continue
		}
		doneChanges, err := c.dnsService.Changes.Create(c.project, managedZone, plannedChanges).Context(c.context).Do()
		if err != nil {
			return nil, err
		}
		updated = append(updated, ToDnsRecords(managedZone, doneChanges)...)
	}
	return updated, nil
}

func changesToUpdateDnsRecordValues(records DnsRecords, newValues []string) *dns.Change {
	changes := &dns.Change{}
	for _, record := range records {
		if reflect.DeepEqual(record.Rrdatas, newValues) {
			continue
		}
		changes.Deletions = append(changes.Deletions, record.ResourceRecordSet)
		addition := *record.ResourceRecordSet
		addition.Rrdatas = newValues
		changes.Additions = append(changes.Additions, &addition)
	}
	if len(changes.Additions) == 0 {
		return nil
	}
	return changes
}

type DnsRecord struct {
	ManagedZone string
	OldRrdatas  []string
	*dns.ResourceRecordSet
}

func (record DnsRecord) NameAndType() string {
	return fmt.Sprintf("%v %v", record.Name, record.Type)
}

type DnsRecords []*DnsRecord

func ToDnsRecords(managedZone string, change *dns.Change) DnsRecords {
	var results DnsRecords
	for _, addition := range change.Additions {
		result := &DnsRecord{ManagedZone: managedZone, ResourceRecordSet: addition}
		for _, deletion := range change.Deletions {
			if deletion.Name == addition.Name {
				result.OldRrdatas = deletion.Rrdatas
			}
		}
		results = append(results, result)
	}
	return results
}

func (records DnsRecords) GroupByZone() map[string]DnsRecords {
	byZone := make(map[string]DnsRecords)
	for _, record := range records {
		byZone[record.ManagedZone] = append(byZone[record.ManagedZone], record)
	}
	return byZone
}

func (records DnsRecords) NamesAndTypes() []string {
	names := make([]string, len(records))
	for i, record := range records {
		names[i] = record.NameAndType()
	}
	return names
}
