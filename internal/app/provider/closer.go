package provider

import "errors"


type closerStack struct {
	fns []func() error
}

func (c *closerStack) Add(fn func()) {
	c.fns = append(c.fns, func() error { fn(); return nil })
}

func (c *closerStack) AddWithError(fn func() error) {
	c.fns = append(c.fns, fn)
}

func (c *closerStack) Close() error {
	var errs []error

	for i := len(c.fns) - 1; i >= 0; i-- {
		if err := c.fns[i](); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
