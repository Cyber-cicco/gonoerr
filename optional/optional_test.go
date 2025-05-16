package optional

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptionalNil(t *testing.T) {
	var nullAbleString *string
	optString := OfNullable(nullAbleString)
	optString.IfPresentOrElse(
		func(s *string) {
			t.Fatalf("optString should be null")
		},
		func() {},
	)
	require.True(t, !optString.IsPresent())
	res := false
    chainRes := Map(optString, func(s *string) bool {
		t := true
		return t
	}).IfPresent(func(b *bool) {
		t.Fatalf("optBool is not supposed to be there")
	}).IfPresentOrElse(
		func(b *bool) {
			t.Fatalf("optBool is not supposed to be there")
		},
		func() {},
	).OrElseDo(func() {
		res = true
	}).OrElseGet(false)

	require.True(t, res)
    require.False(t, chainRes)
    resString := optString.AsResult(errors.New("bonjour"))
    require.True(t, resString.IsErr())


	optPtrString := Of(nullAbleString)
	optPtrString.IfPresentOrElse(
		func(s **string) {},
		func() {
			t.Fatalf("pointer to pointer of string must exists")
		},
	)
}

func TestOptional(t *testing.T) {

	str := "bonjour monde"
	optString := OfNullable(&str)
	optString.IfPresentOrElse(
		func(s *string) {},
		func() {
			t.Fatalf("string is supposed to there")
		},
	)
	require.True(t, optString.IsPresent())
	optString.Filter(
		func(s *string) bool {
			return len(*s) > 0
		},
	).IfPresentOrElse(
		func(s *string) {
		},
		func() {
			t.Fatalf("string is supposed to be there")
		},
	)
    resString := optString.AsResult(errors.New("string is supposed to be there"))
    require.True(t, resString.IsOk())
	res := false
    chainRes := optString.Filter(func(s *string) bool {
		return len(*s) > 0
	}).OrElseDo(func() {
		t.Fatalf("string is supposed to be there")
	}).IfPresent(func(s *string) {
		res = true
	}).Filter(func(s *string) bool {
		return len(*s) > 100
	}).IfPresent(func(s *string) {
		t.Fatalf("string is not supposed to be there")
	}).OrElseGet("hello world")
    require.True(t, res)
    require.Equal(t, "hello world", chainRes)

}
