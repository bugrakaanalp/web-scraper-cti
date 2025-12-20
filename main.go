package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	defaultTimeout = 60 * time.Second
	outputDir      = "output"
)

type ScraperResult struct {
	URL            string
	HTMLFile       string
	ScreenshotFile string
	URLsFile       string
	Success        bool
	Error          error
}

func main() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║          WEB SCRAPER - CTI Görev Programı               ║")
	fmt.Println("║              (Opera GX Version)                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Opera GX yolunu bul
	operaPath := findOperaGX()
	if operaPath == "" {
		fmt.Println("⚠ HATA: Opera GX bulunamadı!")
		fmt.Println("\nOperaBrowser yolu manuel olarak girilmelidir.")
		fmt.Println("Örnek Windows yolu:")
		fmt.Println("C:\\Users\\KullaniciAdi\\AppData\\Local\\Programs\\Opera GX\\opera.exe")
		fmt.Print("\nOpera GX yolunu girin: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		operaPath = strings.TrimSpace(input)

		if _, err := os.Stat(operaPath); os.IsNotExist(err) {
			log.Fatalf("Opera GX bulunamadı: %s", operaPath)
		}
	}

	fmt.Printf("✓ Opera GX bulundu: %s\n\n", operaPath)

	// Output klasörünü oluştur
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Output klasörü oluşturulamadı: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)
	successCount := 0
	failCount := 0

	for {
		fmt.Println("\n" + strings.Repeat("-", 60))
		fmt.Print("Lütfen scrape edilecek URL'i girin (çıkmak için 'q'): ")

		input, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Okuma hatası: %v", err)
			continue
		}

		// Girdiyi temizle
		targetURL := strings.TrimSpace(input)

		// Çıkış kontrolü
		if strings.ToLower(targetURL) == "q" || strings.ToLower(targetURL) == "quit" || strings.ToLower(targetURL) == "exit" {
			fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
			fmt.Printf("║  Toplam İşlem: %d | Başarılı: %d | Başarısız: %d         \n", successCount+failCount, successCount, failCount)
			fmt.Println("╚══════════════════════════════════════════════════════════╝")
			fmt.Println("✓ Program sonlandırılıyor. İyi günler!")
			break
		}

		// Boş giriş kontrolü
		if targetURL == "" {
			fmt.Println("⚠ Hata: URL boş olamaz!")
			continue
		}

		// URL validasyonu
		if err := validateURL(targetURL); err != nil {
			fmt.Printf("⚠ Geçersiz URL: %v\n", err)
			failCount++
			continue
		}

		fmt.Printf("\n▶ Scraping başlatılıyor: %s\n", targetURL)
		fmt.Println("⏳ Lütfen bekleyin...")

		// scraping işlemini başlat
		result := scrapeWebsite(targetURL, operaPath)

		// sonuçlar
		printResults(result)

		if result.Success {
			successCount++
		} else {
			failCount++
		}

		// tamam mı devam mı
		fmt.Print("\nBaşka bir site scrape etmek ister misiniz? (e/h): ")
		continueInput, _ := reader.ReadString('\n')
		continueInput = strings.ToLower(strings.TrimSpace(continueInput))

		if continueInput == "h" || continueInput == "hayır" || continueInput == "n" || continueInput == "no" {
			fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
			fmt.Printf("║  Toplam İşlem: %d | Başarılı: %d | Başarısız: %d         \n", successCount+failCount, successCount, failCount)
			fmt.Println("╚══════════════════════════════════════════════════════════╝")
			fmt.Println("✓ Program sonlandırılıyor. İyi günler!")
			break
		}
	}
}

func findOperaGX() string {
	// Windows için Opera GX varsayılan yolları
	possiblePaths := []string{
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Opera GX", "opera.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES"), "Opera GX", "opera.exe"),
		filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Opera GX", "opera.exe"),
		"C:\\Program Files\\Opera GX\\opera.exe",
		"C:\\Program Files (x86)\\Opera GX\\opera.exe",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func validateURL(rawURL string) error {
	// http veya https yoksa ekle
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL ayrıştırılamadı: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL http veya https ile başlamalıdır")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("URL geçerli bir host içermelidir")
	}

	return nil
}

func scrapeWebsite(targetURL string, operaPath string) ScraperResult {
	result := ScraperResult{
		URL:     targetURL,
		Success: false,
	}

	// URL'yi düzelt (http/https yoksa ekle)
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
		result.URL = targetURL
	}

	// Dosya adları için güvenli isim oluştur
	safeName := sanitizeFilename(targetURL)
	timestamp := time.Now().Format("20060102_150405")

	result.HTMLFile = filepath.Join(outputDir, fmt.Sprintf("%s_%s.html", safeName, timestamp))
	result.ScreenshotFile = filepath.Join(outputDir, fmt.Sprintf("%s_%s.png", safeName, timestamp))
	result.URLsFile = filepath.Join(outputDir, fmt.Sprintf("%s_%s_urls.txt", safeName, timestamp))

	// opera gx için chrome context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath(operaPath),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.WindowSize(1920, 1080),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancel()

	// timeout ayarla
	ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var htmlContent string
	var screenshotBuf []byte
	var links []string

	// siteyi yükle ve işlemleri yap
	err := chromedp.Run(ctx,
		chromedp.Navigate(targetURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(5*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
		chromedp.FullScreenshot(&screenshotBuf, 90),
		chromedp.Evaluate(`Array.from(document.querySelectorAll('a[href]')).map(a => a.href).filter(href => href.startsWith('http'))`, &links),
	)

	if err != nil {
		result.Error = fmt.Errorf("sayfa yüklenemedi: %w", err)
		return result
	}

	// HTML içeriğini kaydet
	if err := os.WriteFile(result.HTMLFile, []byte(htmlContent), 0644); err != nil {
		result.Error = fmt.Errorf("HTML dosyası kaydedilemedi: %w", err)
		return result
	}
	fmt.Printf("  ✓ HTML içeriği kaydedildi (%d bytes)\n", len(htmlContent))

	// ss kaydet
	if err := os.WriteFile(result.ScreenshotFile, screenshotBuf, 0644); err != nil {
		result.Error = fmt.Errorf("ekran görüntüsü kaydedilemedi: %w", err)
		return result
	}
	fmt.Printf("  ✓ Ekran görüntüsü kaydedildi (%d KB)\n", len(screenshotBuf)/1024)

	// url kaydet
	uniqueCount, err := saveURLs(result.URLsFile, links)
	if err != nil {
		result.Error = fmt.Errorf("URL listesi kaydedilemedi: %w", err)
		return result
	}
	fmt.Printf("  ✓ URL listesi kaydedildi (%d benzersiz link)\n", uniqueCount)

	result.Success = true
	return result
}

func sanitizeFilename(rawURL string) string {
	parsedURL, _ := url.Parse(rawURL)
	name := parsedURL.Host
	if parsedURL.Path != "" && parsedURL.Path != "/" {
		name += strings.ReplaceAll(parsedURL.Path, "/", "_")
	}

	// şu karakterleri sil
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "&", "_")
	name = strings.ReplaceAll(name, "=", "_")
	name = strings.ReplaceAll(name, ".", "_")

	// length kontrol
	if len(name) > 50 {
		name = name[:50]
	}

	return name
}

func saveURLs(filename string, links []string) (int, error) {
	// url filtrele
	uniqueLinks := make(map[string]bool)
	for _, link := range links {
		if link != "" && (strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://")) {
			uniqueLinks[link] = true
		}
	}

	// dosyaya yaz
	f, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	f.WriteString(fmt.Sprintf("Toplam Benzersiz URL: %d\n", len(uniqueLinks)))
	f.WriteString(fmt.Sprintf("Tarih: %s\n\n", time.Now().Format("02/01/2006 15:04:05")))
	f.WriteString(strings.Repeat("=", 70) + "\n\n")

	count := 0
	for link := range uniqueLinks {
		count++
		f.WriteString(fmt.Sprintf("%d. %s\n", count, link))
	}

	return len(uniqueLinks), nil
}

func printResults(result ScraperResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))

	if result.Success {
		fmt.Println("✓✓✓ İŞLEM BAŞARILI ✓✓✓")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("URL           : %s\n", result.URL)
		fmt.Printf("HTML Dosyası  : %s\n", result.HTMLFile)
		fmt.Printf("Screenshot    : %s\n", result.ScreenshotFile)
		fmt.Printf("URL Listesi   : %s\n", result.URLsFile)
	} else {
		fmt.Println("✗✗✗ İŞLEM BAŞARISIZ ✗✗✗")
		fmt.Println(strings.Repeat("=", 60))
		fmt.Printf("URL   : %s\n", result.URL)
		fmt.Printf("Hata  : %v\n", result.Error)
		fmt.Println("\nOlası Sebepler:")
		fmt.Println("  • İnternet bağlantısı sorunu")
		fmt.Println("  • Site erişilebilir değil")
		fmt.Println("  • Timeout süresi aşıldı")
		fmt.Println("  • Geçersiz URL formatı")
		fmt.Println("  • Opera GX yolu yanlış")
	}

	fmt.Println(strings.Repeat("=", 60))
}
