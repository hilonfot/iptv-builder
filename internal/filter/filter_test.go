package filter

import (
	"testing"

	"github.com/hilonfot/iptv-builder/internal/model"
)

func makeCh(names ...string) []*model.Channel {
	ch := make([]*model.Channel, len(names))
	for i, n := range names {
		ch[i] = &model.Channel{Name: n, Canonical: n, Valid: true}
	}
	return ch
}

func TestFilter_BasicKeep(t *testing.T) {
	ch := makeCh("CCTV1", "购物频道", "湖南卫视", "广告频道")
	keep := []string{"CCTV", "湖南卫视"}
	exclude := []string{"购物", "广告"}

	result := Filter(ch, exclude, keep)

	if len(result) != 2 {
		t.Fatalf("got %d, want 2", len(result))
	}
	if result[0].Name != "CCTV1" {
		t.Errorf("result[0].Name = %q, want CCTV1", result[0].Name)
	}
	if result[1].Name != "湖南卫视" {
		t.Errorf("result[1].Name = %q, want 湖南卫视", result[1].Name)
	}
}

func TestFilter_ExcludeFirst(t *testing.T) {
	// A channel matching both keep and exclude should be excluded.
	ch := makeCh("CCTV购物")
	keep := []string{"CCTV"}
	exclude := []string{"购物"}

	result := Filter(ch, exclude, keep)

	if len(result) != 0 {
		t.Fatalf("got %d, want 0 (exclude takes priority)", len(result))
	}
}

func TestFilter_EmptyExclude(t *testing.T) {
	ch := makeCh("CCTV1", "湖南卫视", "浙江卫视")
	keep := []string{"CCTV", "卫视"}

	result := Filter(ch, nil, keep)
	if len(result) != 3 {
		t.Fatalf("got %d, want 3", len(result))
	}
}

func TestFilter_EmptyKeep(t *testing.T) {
	ch := makeCh("CCTV1", "湖南卫视")
	// If keep is empty, nothing is retained after we pass this module.
	result := Filter(ch, []string{"广告"}, []string{})
	if len(result) != 0 {
		t.Fatalf("got %d, want 0", len(result))
	}
}

func TestFilter_NothingExcluded(t *testing.T) {
	ch := makeCh("CCTV1", "湖南卫视")
	keep := []string{"CCTV", "卫视"}
	exclude := []string{"购物"}

	result := Filter(ch, exclude, keep)
	if len(result) != 2 {
		t.Fatalf("got %d, want 2", len(result))
	}
}

func TestFilter_SubstringMatch(t *testing.T) {
	ch := makeCh("CCTV1综合HD")
	keep := []string{"CCTV"}

	result := Filter(ch, nil, keep)
	if len(result) != 1 {
		t.Fatalf("got %d, want 1", len(result))
	}
}

func TestFilter_CanonicalMatch(t *testing.T) {
	ch := []*model.Channel{{Name: "CCTV1HD", Canonical: "CCTV1", Valid: true}}
	keep := []string{"CCTV"}

	result := Filter(ch, nil, keep)
	if len(result) != 1 {
		t.Fatalf("got %d, want 1", len(result))
	}
}

func TestFilter_ExcludeCanonicalMatch(t *testing.T) {
	ch := []*model.Channel{{Name: "xx", Canonical: "购物频道", Valid: true}}
	exclude := []string{"购物"}

	result := Filter(ch, exclude, []string{"xx"})
	if len(result) != 0 {
		t.Fatalf("got %d, want 0", len(result))
	}
}

func TestFilter_NilChannel(t *testing.T) {
	ch := []*model.Channel{nil, {Name: "CCTV1", Canonical: "CCTV1", Valid: true}}
	result := Filter(ch, nil, []string{"CCTV"})
	if len(result) != 1 {
		t.Fatalf("got %d, want 1", len(result))
	}
}

func TestFilter_EmptyList(t *testing.T) {
	result := Filter(nil, []string{"购物"}, []string{"CCTV"})
	if len(result) != 0 {
		t.Errorf("got %d, want 0", len(result))
	}
}

func TestFilter_AllExcluded(t *testing.T) {
	ch := makeCh("购物1", "广告2", "导视3")
	exclude := []string{"购物", "广告", "导视"}
	keep := []string{"CCTV"}

	result := Filter(ch, exclude, keep)
	if len(result) != 0 {
		t.Fatalf("got %d, want 0", len(result))
	}
}
