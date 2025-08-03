package app_test

import (
	"testing"
	"time"

	"github.com/luckygeck/lai/app"

	"fyne.io/fyne/v2/test"
)

func TestSmokeTest(t *testing.T) {
	testApp := test.NewApp()
	a := app.New(testApp)

	go a.Run()
	time.Sleep(time.Millisecond * 100)

	testApp.Quit()
}
