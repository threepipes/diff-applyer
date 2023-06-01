package editdist

type Command int

const (
	Unknown Command = iota
	Ins
	Del
	Rpl
	Ign
	End
)

func (c Command) String() string {
	switch c {
	case Unknown:
		return "Unknown"
	case Ins:
		return "Ins"
	case Del:
		return "Del"
	case Rpl:
		return "Replace"
	case Ign:
		return "Ignore"
	case End:
		return "End"
	}
	return "Invalid"
}

type Edit struct {
	Cmd  Command
	Word string
}

const (
	InsCost = 1
	DelCost = 1
	RplCost = 1
)

func WordBased(src []string, tgt []string) []Edit {
	maxCost := len(src)*DelCost + len(tgt)*InsCost
	dp := make([][]int, len(src)+1)
	cmd := make([][]Command, len(src)+1)
	for i := 0; i < len(src)+1; i++ {
		dp[i] = make([]int, len(tgt)+1)
		cmd[i] = make([]Command, len(tgt)+1)
		for j := 0; j < len(tgt)+1; j++ {
			dp[i][j] = maxCost
		}
	}

	for i := 0; i < len(src)+1; i++ {
		dp[i][0] = i
		cmd[i][0] = Del
	}
	for i := 0; i < len(tgt)+1; i++ {
		dp[0][i] = i
		cmd[0][i] = Ins
	}
	cmd[0][0] = End

	for i := 1; i < len(src)+1; i++ {
		for j := 1; j < len(tgt)+1; j++ {
			del := dp[i-1][j] + DelCost
			ins := dp[i][j-1] + InsCost
			rplCost := RplCost
			if src[i-1] == tgt[j-1] {
				rplCost = 0
			}
			rpl := dp[i-1][j-1] + rplCost
			switch {
			case del <= ins && del <= rpl:
				dp[i][j] = del
				cmd[i][j] = Del
			case ins <= del && ins <= rpl:
				dp[i][j] = ins
				cmd[i][j] = Ins
			default:
				dp[i][j] = rpl
				if rplCost == 0 {
					cmd[i][j] = Ign
				} else {
					cmd[i][j] = Rpl
				}
			}
		}
	}

	edt := make([]Edit, 0, len(src))
	y := len(src)
	x := len(tgt)
loop:
	for {
		switch cmd[y][x] {
		case Ins:
			edt = append(edt, Edit{
				Cmd:  Ins,
				Word: tgt[x-1],
			})
			x--
		case Del:
			edt = append(edt, Edit{
				Cmd: Del,
			})
			y--
		case Rpl:
			edt = append(edt, Edit{
				Cmd:  Rpl,
				Word: tgt[x-1],
			})
			x--
			y--
		case Ign:
			edt = append(edt, Edit{
				Cmd: Ign,
			})
			x--
			y--
		case End:
			break loop
		default:
			panic("Unknown command found during the cmd back tracking")
		}
	}

	for i := 0; i < len(edt)/2; i++ {
		j := len(edt) - 1 - i
		edt[i], edt[j] = edt[j], edt[i]
	}
	return edt
}
