package faker

import (
	"hash/fnv"
	"math/rand"
	"strconv"
	"time"

	"github.com/monitoror/monitoror/pkg/contributions"
)

// Contributions returns a stable fake contributions calendar for a user: same user, same values.
func Contributions(user string, weeks int) []string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(user))
	random := rand.New(rand.NewSource(int64(hash.Sum64()))) //nolint:gosec

	days := contributions.Days(time.Now(), weeks)
	values := make([]string, len(days))
	for i, day := range days {
		count := 0
		weekend := day.Weekday() == time.Sunday || day.Weekday() == time.Saturday
		if (!weekend && random.Intn(10) < 7) || (weekend && random.Intn(10) < 2) {
			count = 1 + random.Intn(12)
		}
		values[i] = strconv.Itoa(count)
	}

	return values
}
