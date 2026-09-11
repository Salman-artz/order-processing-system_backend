package repository

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

// Float64ToNumeric converts a float64 to pgtype.Numeric
func Float64ToNumeric(f float64) pgtype.Numeric {
	// simple conversion
	s := fmt.Sprintf("%.2f", f)
	n := pgtype.Numeric{}
	n.Scan(s)
	return n
}

// NumericToFloat64 converts a pgtype.Numeric to float64
func NumericToFloat64(n pgtype.Numeric) float64 {
	val, _ := n.Float64Value()
	return val.Float64
}
