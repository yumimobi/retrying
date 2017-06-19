# retrying
Retrying is a golang binding of python retrying library https://github.com/rholder/retrying

[![Build Status](https://travis-ci.org/yumimobi/retrying.svg?branch=master)](https://travis-ci.org/yumimobi/retrying)
[![Go Report Card](http://goreportcard.com/badge/yumimobi/retrying)](http://goreportcard.com/report/yumimobi/retrying)
[![codecov](https://codecov.io/gh/yumimobi/retrying/branch/master/graph/badge.svg)](https://codecov.io/gh/yumimobi/retrying)


## Installation

```bash
$ go get github.com/yumimobi/retrying
```

## Usage

```go
package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/yumimobi/retrying"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	err := retrying.New().
		Stack(2048, true).
		MaxAttemptTimes(5).
		Function(func() error {
			if rand.Int63n(100) > 80 {
				return nil
			}
			return fmt.Errorf("dllm")
		}).
		WaitFixed(time.Second).
		WaitRandom(time.Second, time.Second*3).
		MaxDelay(time.Minute).
		Try()

	fmt.Println(err == nil)
}
```
