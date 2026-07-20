package transform

import (
	"context"
	"testing"
	"time"

	"github.com/keeleysam/terraform-burnham/internal/provider/resbudget"
)

// 1000 records is a large-but-plausible decoded input for the query engines.
// jsonata allocates several times more than jq/jmespath for the same work, hence
// its larger budget.
func TestResourceBudget(t *testing.T) {
	ctx := context.Background()
	data := benchData(1000)

	resbudget.Check(t, "jq_query(1000 records)", 4<<20, time.Second, func() {
		if _, err := runJQ(ctx, data, `.records[] | select(.enabled) | .name`, nil); err != nil {
			t.Fatal(err)
		}
	})
	resbudget.Check(t, "jmespath(1000 records)", 4<<20, time.Second, func() {
		if _, err := runJMESPath(data, `records[?enabled].name`); err != nil {
			t.Fatal(err)
		}
	})
	resbudget.Check(t, "jsonata_query(1000 records)", 16<<20, 2*time.Second, func() {
		if _, err := runJSONata(ctx, data, `records[enabled].name`); err != nil {
			t.Fatal(err)
		}
	})
}
