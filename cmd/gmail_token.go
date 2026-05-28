package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

var (
	gmailTokenCreds string
	gmailTokenOut   string
	gmailTokenPort  int
	gmailTokenNoOpen bool
)

var gmailTokenCmd = &cobra.Command{
	Use:   "gmail-token",
	Short: "One-off OAuth bootstrap that exchanges Google consent for token.json (refresh-token included)",
	Long: `Reads the OAuth client JSON downloaded from Google Cloud Console
(--credentials), opens a consent URL in your browser, captures the auth code
on a local callback, and writes a token.json with the long-lived refresh
token that 'watch'/'campaign' need to talk to Gmail.

Run this once per project (then point NURIKUN_GMAIL_TOKEN at the file).
The local callback listens on http://localhost:<port>/callback — that URL
must be registered as an authorized redirect URI on the OAuth client.

The scopes requested match what the Gmail mailbox provider needs:
gmail.readonly, gmail.modify, gmail.send.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if gmailTokenCreds == "" {
			return fmt.Errorf("--credentials is required (path to Google OAuth client JSON)")
		}
		if gmailTokenOut == "" {
			return fmt.Errorf("--out is required (path to write token.json)")
		}
		return runGmailTokenBootstrap(gmailTokenCreds, gmailTokenOut, gmailTokenPort, gmailTokenNoOpen)
	},
}

func runGmailTokenBootstrap(credsPath, outPath string, port int, noOpen bool) error {
	raw, err := os.ReadFile(credsPath)
	if err != nil {
		return fmt.Errorf("read credentials: %w", err)
	}
	cfg, err := google.ConfigFromJSON(raw,
		gmail.GmailReadonlyScope,
		gmail.GmailModifyScope,
		gmail.GmailSendScope,
	)
	if err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}
	cfg.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", port)

	state := fmt.Sprintf("musu-nurikun-%d", time.Now().UnixNano())
	// AccessTypeOffline + ApprovalForce ensures a refresh_token is returned,
	// even if the user previously consented without offline access.
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	type result struct {
		code string
		err  error
	}
	resCh := make(chan result, 1)

	mux := http.NewServeMux()
	// Root catch-all so any redirect path (/, /callback, /oauth2callback, etc.)
	// reaches us. Google's redirect path can vary with how the OAuth client
	// was registered in Cloud Console (Web vs Desktop), and silently 404-ing
	// is a frustrating dead-end. Ignore noise like favicon.ico.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		fmt.Printf("…callback hit: %s %s?%s\n", r.Method, r.URL.Path, r.URL.RawQuery)

		// Browser noise — don't treat as auth callback.
		if r.URL.Path == "/favicon.ico" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if errStr := q.Get("error"); errStr != "" {
			http.Error(w, "oauth error: "+errStr, http.StatusBadRequest)
			resCh <- result{err: fmt.Errorf("oauth error: %s", errStr)}
			return
		}
		code := q.Get("code")
		if code == "" {
			// Not the OAuth redirect — could be a stray probe. Show a hint
			// and keep waiting rather than ending the flow.
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "Waiting for OAuth callback (path %q has no ?code= yet).\n", r.URL.Path)
			return
		}
		if q.Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			resCh <- result{err: fmt.Errorf("state mismatch")}
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "OAuth consent received. You can close this tab and return to the terminal.")
		resCh <- result{code: code}
	})

	// Bind synchronously so port-conflict errors surface immediately and don't
	// silently leave the consent URL pointing at another service. Bind to
	// :PORT (dual-stack) so both 127.0.0.1 and ::1 reach us — on Windows
	// "localhost" sometimes resolves to ::1 first.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("bind port %d: %w — another service is likely using it; try --port <other>", port, err)
	}
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			resCh <- result{err: fmt.Errorf("callback server: %w", err)}
		}
	}()
	fmt.Printf("🌐 callback server listening on %s\n", listener.Addr())

	fmt.Printf("🔑 Open this URL to grant Gmail access:\n\n%s\n\n", authURL)
	if !noOpen {
		if err := openBrowser(authURL); err != nil {
			fmt.Printf("(could not auto-open browser: %v — copy the URL above manually)\n", err)
		}
	}
	fmt.Printf("…waiting on http://localhost:%d/callback (5 min timeout)\n", port)

	select {
	case res := <-resCh:
		_ = srv.Shutdown(context.Background())
		if res.err != nil {
			return res.err
		}
		tok, err := cfg.Exchange(context.Background(), res.code)
		if err != nil {
			return fmt.Errorf("exchange: %w", err)
		}
		if tok.RefreshToken == "" {
			return fmt.Errorf("no refresh_token returned — revoke the prior consent at https://myaccount.google.com/permissions and rerun this command")
		}
		data, err := json.MarshalIndent(tok, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal token: %w", err)
		}
		if err := os.WriteFile(outPath, data, 0o600); err != nil {
			return fmt.Errorf("write token: %w", err)
		}
		fmt.Printf("\n✅ Saved %s\n", outPath)
		fmt.Println("   Next:")
		fmt.Println("     set NURIKUN_GMAIL_CREDENTIALS to the credentials JSON path")
		fmt.Println("     set NURIKUN_GMAIL_TOKEN to the token path above")
		fmt.Println("     run `musu-nurikun doctor` to verify reachability")
		return nil
	case <-time.After(5 * time.Minute):
		_ = srv.Shutdown(context.Background())
		return fmt.Errorf("timed out waiting for OAuth callback")
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		// NOTE: do NOT use `cmd /c start "" <url>` here — cmd.exe re-parses
		// the command line and treats unquoted `&` in OAuth URLs (which
		// separate query parameters) as command separators, truncating the
		// URL and producing "invalid_request: Required parameter is missing"
		// errors from Google. rundll32 passes the URL straight to the shell
		// URL handler with no re-parsing.
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func init() {
	gmailTokenCmd.Flags().StringVar(&gmailTokenCreds, "credentials", "", "Path to Google OAuth client JSON (required)")
	gmailTokenCmd.Flags().StringVar(&gmailTokenOut, "out", "", "Path to write token.json (required)")
	gmailTokenCmd.Flags().IntVar(&gmailTokenPort, "port", 8080, "Local callback port (must match an authorized redirect URI on the OAuth client)")
	gmailTokenCmd.Flags().BoolVar(&gmailTokenNoOpen, "no-open", false, "Print the consent URL only; do not auto-open a browser")
	rootCmd.AddCommand(gmailTokenCmd)
}
