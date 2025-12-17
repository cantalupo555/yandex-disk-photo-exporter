// Package download handles file download operations on Yandex Disk.
package download

import (
	"context"

	"github.com/chromedp/chromedp"
)

// ClickDownloadButton finds and clicks the Download button.
func ClickDownloadButton(ctx context.Context) error {
	return chromedp.Run(ctx,
		chromedp.Evaluate(`
			(function() {
				const buttons = document.querySelectorAll('button, [role="button"]');
				for (const btn of buttons) {
					const text = btn.textContent?.trim() || '';
					const ariaLabel = btn.getAttribute('aria-label') || '';
					const title = btn.getAttribute('title') || '';
					
					if (text === 'Download' || 
						text === 'Скачать' ||
						ariaLabel.includes('Download') ||
						ariaLabel.includes('Скачать') ||
						title.includes('Download')) {
						btn.click();
						return 'clicked';
					}
				}
				return 'not found';
			})()
		`, nil),
	)
}
