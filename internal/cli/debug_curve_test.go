package cli

import "testing"

func TestDebugDecayCurve(t *testing.T) {
	curve, err := buildDecayCurve("front-loaded", 48, 336)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	for i := 0; i < 15; i++ {
		t.Logf("idx=%d x=%v impact=%v", i, curve.Samples[i].X, curve.Samples[i].Impact)
	}
}
