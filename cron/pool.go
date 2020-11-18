package cron

type pool struct {
	workers chan bool
}

func newPool(capacity int) *pool {
	p := &pool{}
	p.workers = make(chan bool, capacity)
	for i := 0; i < capacity; i++ {
		p.releaseWorker()
	}
	return p
}

func (p *pool) releaseWorker() {
	p.workers <- true
}

func (p *pool) waitForWorker() bool {
	return <-p.workers
}
