package browser

import (
	"math/rand"

	"github.com/playwright-community/playwright-go"
)

type Walker struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	context playwright.BrowserContext
	Project string
	LastPos Point // Keep track of cursor for human-like movement
}

func NewWalker(project string) (*Walker, error) {
	err := playwright.Install()
	if err != nil { return nil, err }

	pw, err := playwright.Run()
	if err != nil { return nil, err }

	// Advanced Stealth: Real Chrome channel and headed mode
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--use-gl=desktop",
		},
	})
	if err != nil { return nil, err }

	// Dynamic User-Agent Pool
	uaPool := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	}
	ua := uaPool[rand.Intn(len(uaPool))]

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(ua),
		Viewport: &playwright.Size{
			Width:  1920 + rand.Intn(100),
			Height: 1080 + rand.Intn(100),
		},
	})
	if err != nil { return nil, err }

	// Inject 2025-standard Anti-Detect scripts
	antiDetectScript := `
		// 1. Mask navigator.webdriver
		Object.defineProperty(navigator, 'webdriver', {get: () => undefined});

		// 2. Spoof WebGL (NVIDIA/Real Hardware)
		const getParameter = WebGLRenderingContext.prototype.getParameter;
		WebGLRenderingContext.prototype.getParameter = function(parameter) {
			if (parameter === 37445) return 'NVIDIA Corporation';
			if (parameter === 37446) return 'NVIDIA GeForce RTX 4080/PCIe/SSE2';
			return getParameter.apply(this, arguments);
		};

		// 3. Canvas Noise (Defeat deterministic hashing)
		const originalGetImageData = CanvasRenderingContext2D.prototype.getImageData;
		CanvasRenderingContext2D.prototype.getImageData = function(x, y, w, h) {
			const imageData = originalGetImageData.apply(this, arguments);
			imageData.data[0] = imageData.data[0] + (Math.random() > 0.5 ? 1 : -1);
			return imageData;
		};
	`
	// Fixed: Proper signature for AddInitScript on context using playwright.Script
	err = context.AddInitScript(playwright.Script{
		Content: playwright.String(antiDetectScript),
	})
	if err != nil { return nil, err }

	return &Walker{
		pw:      pw,
		browser: browser,
		context: context,
		Project: project,
		LastPos: Point{X: 0, Y: 0},
	}, nil
}

func (w *Walker) Navigate(url string) (playwright.Page, error) {
	page, err := w.context.NewPage()
	if err != nil { return nil, err }
	
	if _, err := page.Goto(url); err != nil {
		page.Close()
		return nil, err
	}
	return page, nil
}

func (w *Walker) HumanClick(page playwright.Page, selector string) error {
	element := page.Locator(selector).First()
	box, err := element.BoundingBox()
	if err != nil { return err }

	target := Point{
		X: box.X + box.Width/2 + (rand.Float64()*10 - 5),
		Y: box.Y + box.Height/2 + (rand.Float64()*10 - 5),
	}

	if err := MoveMouseHumanLike(page.Mouse(), w.LastPos, target); err != nil { return err }
	w.LastPos = target
	
	return page.Mouse().Click(target.X, target.Y)
}

func (w *Walker) Close() {
	if w.context != nil { w.context.Close() }
	if w.browser != nil { w.browser.Close() }
	if w.pw != nil { w.pw.Stop() }
}
