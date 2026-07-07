package dynamostore_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/scttfrdmn/prism/pkg/seam"
	"github.com/scttfrdmn/prism/pkg/seam/dynamostore"
	"github.com/scttfrdmn/prism/pkg/seam/seamtest"
)

// The conformance run drives the store against an in-memory fake of the four DynamoDB calls it
// makes. The fake faithfully emulates the single-table semantics the store depends on — pk+sk
// keys, Query-by-pk, and DeleteItem ReturnValues=ALL_OLD (so a missing delete is detectable) —
// so the store's encoding (pk = scope.Key(), sk = id, body = JSON) is exercised exactly as it
// ships. That encoding is byte-identical to prp's dynamostore: the shared-state contract (§4).
//
// (A real-SDK run against the substrate emulator, as prp does, would require bumping prism's
// substrate dependency forward; the fake keeps this convergence dependency-light while still
// proving the request/response handling.)
func TestConformance(t *testing.T) {
	seamtest.RunConformance(t, func(t *testing.T) seam.Store[seamtest.Record] {
		return dynamostore.New[seamtest.Record](newFakeDDB(), "prism-conformance")
	})
}

// fakeDDB is an in-memory stand-in for the DynamoDB API. Items are keyed by (pk, sk) strings.
type fakeDDB struct {
	items map[string]map[string]map[string]types.AttributeValue // pk -> sk -> item
}

func newFakeDDB() *fakeDDB {
	return &fakeDDB{items: map[string]map[string]map[string]types.AttributeValue{}}
}

func s(av types.AttributeValue) string {
	if v, ok := av.(*types.AttributeValueMemberS); ok {
		return v.Value
	}
	return ""
}

func (f *fakeDDB) GetItem(_ context.Context, in *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	pk, sk := s(in.Key["pk"]), s(in.Key["sk"])
	if byPK, ok := f.items[pk]; ok {
		if item, ok := byPK[sk]; ok {
			return &dynamodb.GetItemOutput{Item: item}, nil
		}
	}
	return &dynamodb.GetItemOutput{Item: nil}, nil
}

func (f *fakeDDB) PutItem(_ context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	pk, sk := s(in.Item["pk"]), s(in.Item["sk"])
	if f.items[pk] == nil {
		f.items[pk] = map[string]map[string]types.AttributeValue{}
	}
	f.items[pk][sk] = in.Item
	return &dynamodb.PutItemOutput{}, nil
}

func (f *fakeDDB) DeleteItem(_ context.Context, in *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	pk, sk := s(in.Key["pk"]), s(in.Key["sk"])
	byPK, ok := f.items[pk]
	if !ok {
		return &dynamodb.DeleteItemOutput{}, nil
	}
	old, existed := byPK[sk]
	delete(byPK, sk)
	out := &dynamodb.DeleteItemOutput{}
	if existed {
		// Mirror ReturnValues=ALL_OLD: the store reads Attributes to detect a real deletion.
		out.Attributes = old
	}
	return out, nil
}

func (f *fakeDDB) Query(_ context.Context, in *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	pk := s(in.ExpressionAttributeValues[":pk"])
	var items []map[string]types.AttributeValue
	for _, item := range f.items[pk] {
		items = append(items, item)
	}
	return &dynamodb.QueryOutput{Items: items}, nil
}

var _ dynamostore.API = (*fakeDDB)(nil)
