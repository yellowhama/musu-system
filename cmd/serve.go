package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yellowhama/musu-nurikun/internal/compliance"
	"github.com/yellowhama/musu-nurikun/internal/config"
)

var serveAddr string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the web endpoints for one-click unsubscribe and double opt-in confirmation",
	Long: `Serves the endpoints honored by mailbox providers and double opt-in links:

  GET/POST /unsubscribe?email=<e>&sig=<hmac>   one-click opt-out (RFC 8058)
  GET      /confirm?token=<t>                  complete a double opt-in subscription
  GET      /healthz                            liveness

The unsubscribe link is HMAC-signed (NURIKUN_UNSUB_SECRET / unsub_secret) so it
cannot be forged to opt out arbitrary addresses. Set public_base_url so campaigns
embed links that point here.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		project := viper.GetString("project")
		cfg, err := config.Load(project)
		if err != nil {
			return err
		}
		store, err := openStore()
		if err != nil {
			return err
		}
		defer store.Close()

		mux := http.NewServeMux()

		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "ok")
		})

		// One-click unsubscribe — must be signed. Honored for GET and the RFC 8058 POST.
		mux.HandleFunc("/unsubscribe", func(w http.ResponseWriter, r *http.Request) {
			email := r.URL.Query().Get("email")
			sig := r.URL.Query().Get("sig")
			if email == "" || !compliance.VerifyUnsub(email, sig, cfg.UnsubSecret) {
				http.Error(w, "invalid or unsigned unsubscribe link", http.StatusBadRequest)
				return
			}
			if err := store.Unsubscribe(email, "one-click"); err != nil {
				http.Error(w, "unsubscribe failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "%s 수신거부 완료. 더 이상 메일을 받지 않습니다.\nYou have been unsubscribed.\n", email)
		})

		// Complete a double opt-in subscription via emailed link.
		mux.HandleFunc("/confirm", func(w http.ResponseWriter, r *http.Request) {
			token := r.URL.Query().Get("token")
			if token == "" {
				http.Error(w, "missing token", http.StatusBadRequest)
				return
			}
			sub, err := store.ConfirmSubscriber(token)
			if err != nil {
				http.Error(w, "invalid or already-used confirmation token", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "구독 확인 완료 (%s). 이제부터 메일을 받으십니다.\nSubscription confirmed.\n", sub.Email)
		})

		fmt.Printf("🌐 serving unsubscribe/confirm on %s (project %q)\n", serveAddr, project)
		srv := &http.Server{
			Addr:              serveAddr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		}
		return srv.ListenAndServe()
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", ":8088", "address to listen on")
	rootCmd.AddCommand(serveCmd)
}
