package monitor

import (
	"context"

	"gopkg.in/dfsr.v0/core"
)

func connections(domain *core.Domain) (output []*core.Backlog) {
	for gi := 0; gi < len(domain.Groups); gi++ {
		group := &domain.Groups[gi]

		for mi := 0; mi < len(group.Members); mi++ {
			member := &group.Members[mi]
			to := member.Computer.Host
			if to == "" {
				continue
			}

			for ci := 0; ci < len(member.Connections); ci++ {
				conn := &member.Connections[ci]
				from := conn.Computer.Host
				if from == "" {
					continue
				}
				if !conn.Enabled {
					continue
				}

				output = append(output, &core.Backlog{
					Group: group,
					From:  from,
					To:    to,
				})
			}
		}
	}
	return
}

func cancelRequested(ctx context.Context) bool {
	if ctx == nil {
		panic("nil context")
	}

	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
