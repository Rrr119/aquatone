package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/michenriksen/aquatone/core"
	"github.com/remeh/sizedwaitgroup"
)

func newScreenshotTestSession(t *testing.T, threads int, rate int) (*core.Session, func()) {
	t.Helper()

	outDir, err := ioutil.TempDir("", "aquatone-screenshot-test")
	if err != nil {
		t.Fatal(err)
	}

	screenshotThreads := threads
	screenshotRate := rate
	screenshotTimeout := 1000
	globalThreads := threads
	resolution := "1440,900"
	proxy := ""
	chromePath := ""

	options := core.Options{
		Threads:           &globalThreads,
		OutDir:            &outDir,
		Proxy:             &proxy,
		ChromePath:        &chromePath,
		Resolution:        &resolution,
		ScreenshotTimeout: &screenshotTimeout,
		ScreenshotThreads: &screenshotThreads,
		ScreenshotRate:    &screenshotRate,
	}

	logger := &core.Logger{}
	logger.SetSilent(true)

	session := &core.Session{
		Options: options,
		Out:     logger,
		Stats:   &core.Stats{StartedAt: time.Now()},
		Pages:   make(map[string]*core.Page),
	}
	session.WaitGroup = sizedwaitgroup.New(threads)

	cleanup := func() {
		os.RemoveAll(outDir)
	}
	return session, cleanup
}

func Test500TargetScreenshotsAreDedupedAndReportCompatible(t *testing.T) {
	const targetCount = 500
	const duplicateEvery = 10
	const screenshotThreads = 8

	session, cleanup := newScreenshotTestSession(t, screenshotThreads, 0)
	defer cleanup()

	screenshotter := NewURLScreenshotter()
	screenshotter.session = session
	screenshotter.initializeScheduler()

	var mutex sync.Mutex
	active := 0
	maxActive := 0
	screenshotCount := 0
	screenshotPaths := make(map[string]struct{})
	screenshotter.runScreenshot = func(ctx context.Context, page *core.Page, path string) (bool, error) {
		mutex.Lock()
		active++
		screenshotCount++
		if active > maxActive {
			maxActive = active
		}
		screenshotPaths[path] = struct{}{}
		mutex.Unlock()

		time.Sleep(time.Millisecond)

		mutex.Lock()
		active--
		mutex.Unlock()
		return false, nil
	}

	for i := 0; i < targetCount; i++ {
		target := fmt.Sprintf("https://target-%03d.example.com/path/%d", i, i)
		if _, err := session.AddPage(target); err != nil {
			t.Fatal(err)
		}

		screenshotter.OnURLResponsive(target)
		if i%duplicateEvery == 0 {
			screenshotter.OnURLResponsive(target)
		}
	}
	session.WaitGroup.Wait()

	mutex.Lock()
	actualCount := screenshotCount
	actualMaxActive := maxActive
	actualPathCount := len(screenshotPaths)
	mutex.Unlock()

	if actualCount != targetCount {
		t.Fatalf("took %d screenshots, want %d", actualCount, targetCount)
	}
	if actualPathCount != targetCount {
		t.Fatalf("wrote %d unique screenshot paths, want %d", actualPathCount, targetCount)
	}
	if actualMaxActive > screenshotThreads {
		t.Fatalf("maximum screenshot concurrency was %d, limit is %d", actualMaxActive, screenshotThreads)
	}
	if session.Stats.ScreenshotSuccessful != targetCount {
		t.Fatalf("successful screenshots = %d, want %d", session.Stats.ScreenshotSuccessful, targetCount)
	}
	if session.Stats.ScreenshotDeduped != targetCount/duplicateEvery {
		t.Fatalf("deduplicated screenshots = %d, want %d", session.Stats.ScreenshotDeduped, targetCount/duplicateEvery)
	}

	for _, page := range session.Pages {
		if !page.HasScreenshot || page.ScreenshotPath == "" {
			t.Fatalf("page %s is missing screenshot report metadata", page.URL)
		}
	}

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	var restored core.Session
	if err := json.Unmarshal(sessionJSON, &restored); err != nil {
		t.Fatalf("session JSON is not readable: %v", err)
	}
	if len(restored.Pages) != targetCount {
		t.Fatalf("restored session has %d pages, want %d", len(restored.Pages), targetCount)
	}

	template, err := ioutil.ReadFile("../static/report_template.html")
	if err != nil {
		t.Fatal(err)
	}
	var report bytes.Buffer
	if err := core.NewReport(session, string(template)).Render(&report); err != nil {
		t.Fatalf("existing report template failed to render: %v", err)
	}
	if report.Len() == 0 {
		t.Fatal("report renderer produced an empty report")
	}
}

func TestScreenshotRateSlotsAreEvenlySpaced(t *testing.T) {
	session, cleanup := newScreenshotTestSession(t, 1, 10)
	defer cleanup()

	screenshotter := NewURLScreenshotter()
	screenshotter.session = session
	screenshotter.initializeScheduler()

	now := time.Unix(100, 0)
	expected := []time.Duration{0, 100 * time.Millisecond, 200 * time.Millisecond, 300 * time.Millisecond}
	for i, want := range expected {
		if got := screenshotter.reserveRateSlot(now); got != want {
			t.Fatalf("rate slot %d = %s, want %s", i, got, want)
		}
	}
}
