package keeper

import (
	"fmt"
	"reflect"
	"testing"

	"cosmossdk.io/math"

	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func Test_calReward(t *testing.T) {

	rateList := []types.Rate{
		{Amount: math.NewInt(300), Rate: 1.50},
		{Amount: math.NewInt(200), Rate: 1.20},
		{Amount: math.NewInt(100), Rate: 1.10},
		{Amount: math.NewInt(0), Rate: 1.00},
	}

	tests := []struct {
		lockedAmt  int64
		lockingAmt int64
		want       int64
	}{
		{
			lockedAmt:  100,
			lockingAmt: 0,
			want:       0,
		},
		{
			lockedAmt:  250,
			lockingAmt: 0,
			want:       0,
		},
		{
			lockedAmt:  0,
			lockingAmt: 100,
			want:       100,
		},
		{
			lockedAmt:  0,
			lockingAmt: 200,
			want:       (100 * 1.0) + (100 * 1.1),
		},
		{
			lockedAmt:  0,
			lockingAmt: 250,
			want:       (100 * 1.0) + (100 * 1.1) + (50 * 1.2),
		},
		{
			lockedAmt:  0,
			lockingAmt: 300,
			want:       (100 * 1.0) + (100 * 1.1) + (100 * 1.2),
		},
		{
			lockedAmt:  0,
			lockingAmt: 350,
			want:       (100 * 1.0) + (100 * 1.1) + (100 * 1.2) + (50 * 1.5),
		},
		{
			lockedAmt:  0,
			lockingAmt: 600,
			want:       (100 * 1.0) + (100 * 1.1) + (100 * 1.2) + (300 * 1.5),
		},
		{
			lockedAmt:  100,
			lockingAmt: 100,
			want:       (100 * 1.1),
		},
		{
			lockedAmt:  200,
			lockingAmt: 100,
			want:       (100 * 1.2),
		},
		{
			lockedAmt:  150,
			lockingAmt: 150,
			want:       (50 * 1.1) + (100 * 1.2),
		},
		{
			lockedAmt:  50,
			lockingAmt: 400,
			want:       (50 * 1.0) + (100 * 1.1) + (100 * 1.2) + (150 * 1.5),
		},
	}
	for _, tt := range tests {

		name := fmt.Sprintf("%d adds %d", tt.lockedAmt, tt.lockingAmt)
		oldLock := math.NewInt(tt.lockedAmt)
		newLock := math.NewInt(tt.lockingAmt)
		want := math.NewInt(tt.want)

		t.Run(name, func(t *testing.T) {
			if got := calculateCredit(rateList, oldLock, newLock); !reflect.DeepEqual(got, want) {
				t.Errorf("calCredits() = %v, want %v", got, tt.want)
			}
		})
	}
}
