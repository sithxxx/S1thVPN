// Злоумышленник может записать ваш зашифрованный пакет и отправить его ещё раз — без этого окна сервер примет его снова.
// Окно помнит последние 1984 номера и принимает перепутанные сетью пакеты, но никогда — повтор.
// Алгоритм — RFC 6479: кольцо из 32 блоков по 64 бита, без сдвигов всего массива.
// Package replay — защита от повторной отправки пакетов (anti-replay).
// Скользящее окно в виде кольца битовых блоков, алгоритм RFC 6479.

package replay

import "sync"

const (
	blockBits  = 64
	ringBlocks = 32
	blockMask  = ringBlocks - 1
	WindowSize = (ringBlocks - 1) * blockBits
)

type Window struct {
	mu   sync.Mutex
	last uint64
	ring [ringBlocks]uint64
}

func (w *Window) Accept(counter, limit uint64) bool {
	if counter >= limit {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	block := counter / blockBits
	if counter > w.last {
		current := w.last / blockBits
		diff := block - current
		if diff > ringBlocks {
			diff = ringBlocks
		}
		for i := uint64(1); i <= diff; i++ {
			w.ring[(current+i)&blockMask] = 0
		}
		w.last = counter
	} else if w.last-counter > WindowSize {
		return false
	}
	bit := uint64(1) << (counter % blockBits)
	idx := block & blockMask
	if w.ring[idx]&bit != 0 {
		return false
	}
	w.ring[idx] |= bit
	return true
}
