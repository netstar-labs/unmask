// Runnable example: detect confusable homographs of a brand set via the UTS-39
// skeleton JOIN, and flag mixed-script labels.
//
//	go run ./example/confusable
package main

import (
	"fmt"

	"github.com/netstar-labs/unmask"
)

func main() {
	brands := []string{"paypal", "google", "amazon"}

	for _, obs := range []string{"pаypаl", "g00gle", "amazon", "paypal-login"} {
		r := unmask.Analyze(obs)
		hit := ""
		for _, b := range brands {
			if unmask.Confusable(obs, b) {
				hit = b
				break
			}
		}
		fmt.Printf("%-14q skeleton=%-14q scripts=%v mixed=%-5v", obs, r.Skeleton, r.Scripts, r.MixedScript)
		if hit != "" {
			fmt.Printf("  ⚠ confusable with %q", hit)
		}
		fmt.Println()
	}
}
