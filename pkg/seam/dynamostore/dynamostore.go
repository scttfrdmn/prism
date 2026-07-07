// Package dynamostore is the DynamoDB-backed seam.Store: the shared cloud-core implementation.
//
// One table, partitioned by Scope: pk = scope.Key() (tenant/project/pi/grant), sk = id, body =
// JSON. This encoding is byte-identical to prism-research-portal's dynamostore — that identity is
// the wire contract that makes shared state work (design §4): a record prp (web) writes to the
// cloud table is the record Prism (desktop, pointed at the same table) reads, and vice versa.
// Neither client needs a sync protocol because there is one table and one encoding.
//
// It depends on a narrow API interface (the four DynamoDB calls it makes), not the concrete
// *dynamodb.Client, so it is unit-tested against an in-memory fake — no live AWS.
//
// NOTE: keep this identical to prism-research-portal/pkg/seam/dynamostore. prp will re-point at
// THIS definition (deleting its duplicate) as the unification step; until then they must not drift.
package dynamostore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/scttfrdmn/prism/pkg/seam"
)

// API is the subset of the DynamoDB client dynamostore uses. *dynamodb.Client satisfies it; the
// conformance test supplies an in-memory fake.
type API interface {
	GetItem(ctx context.Context, in *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, in *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, in *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Query(ctx context.Context, in *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

const (
	attrPK   = "pk"
	attrSK   = "sk"
	attrBody = "body"
)

// Store is a DynamoDB-backed seam.Store[T]. Construct with New.
type Store[T any] struct {
	api   API
	table string
}

// New returns a store over the given table using the given API client.
func New[T any](api API, table string) *Store[T] {
	return &Store[T]{api: api, table: table}
}

func key(scope seam.Scope, id string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{
		attrPK: &types.AttributeValueMemberS{Value: scope.Key()},
		attrSK: &types.AttributeValueMemberS{Value: id},
	}
}

// Get returns the record at (scope, id), or seam.ErrNotFound if none exists.
func (s *Store[T]) Get(ctx context.Context, scope seam.Scope, id string) (T, error) {
	var zero T
	out, err := s.api.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key:       key(scope, id),
	})
	if err != nil {
		return zero, fmt.Errorf("dynamostore get: %w", err)
	}
	if out.Item == nil {
		return zero, seam.ErrNotFound
	}
	return decodeBody[T](out.Item)
}

// List returns all records in scope (empty slice when none), paging through Query results.
func (s *Store[T]) List(ctx context.Context, scope seam.Scope) ([]T, error) {
	out := make([]T, 0)
	var startKey map[string]types.AttributeValue
	for {
		page, err := s.api.Query(ctx, &dynamodb.QueryInput{
			TableName:                 aws.String(s.table),
			KeyConditionExpression:    aws.String("#pk = :pk"),
			ExpressionAttributeNames:  map[string]string{"#pk": attrPK},
			ExpressionAttributeValues: map[string]types.AttributeValue{":pk": &types.AttributeValueMemberS{Value: scope.Key()}},
			ExclusiveStartKey:         startKey,
		})
		if err != nil {
			return nil, fmt.Errorf("dynamostore list: %w", err)
		}
		for _, item := range page.Items {
			v, err := decodeBody[T](item)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		if len(page.LastEvaluatedKey) == 0 {
			break
		}
		startKey = page.LastEvaluatedKey
	}
	return out, nil
}

// Put writes v at (scope, id), storing the value as a JSON body attribute.
func (s *Store[T]) Put(ctx context.Context, scope seam.Scope, id string, v T) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("dynamostore put marshal: %w", err)
	}
	item := key(scope, id)
	item[attrBody] = &types.AttributeValueMemberS{Value: string(body)}
	if _, err := s.api.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item:      item,
	}); err != nil {
		return fmt.Errorf("dynamostore put: %w", err)
	}
	return nil
}

// Delete removes the record at (scope, id), or returns seam.ErrNotFound if none existed.
func (s *Store[T]) Delete(ctx context.Context, scope seam.Scope, id string) error {
	out, err := s.api.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:    aws.String(s.table),
		Key:          key(scope, id),
		ReturnValues: types.ReturnValueAllOld, // lets us detect a missing record
	})
	if err != nil {
		return fmt.Errorf("dynamostore delete: %w", err)
	}
	if len(out.Attributes) == 0 {
		return seam.ErrNotFound
	}
	return nil
}

func decodeBody[T any](item map[string]types.AttributeValue) (T, error) {
	var zero T
	raw, ok := item[attrBody].(*types.AttributeValueMemberS)
	if !ok {
		return zero, fmt.Errorf("dynamostore: item missing %q string attribute", attrBody)
	}
	var v T
	if err := json.Unmarshal([]byte(raw.Value), &v); err != nil {
		return zero, fmt.Errorf("dynamostore unmarshal body: %w", err)
	}
	return v, nil
}

// compile-time check
var _ seam.Store[int] = (*Store[int])(nil)
