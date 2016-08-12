package helper

import (
	"errors"
	"log"

	"github.com/go-ole/go-ole"
)

func makeBacklog(sa *ole.SafeArrayConversion) (backlog []int) {
	values := sa.ToValueArray()

	backlog = make([]int, 0, len(values))

	for i := 0; i < len(values); i++ {
		if val, ok := values[i].(int32); ok {
			backlog = append(backlog, int(val))
		} else {
			log.Fatal(errors.New("invalid backlog value"))
		}
	}

	return
}
