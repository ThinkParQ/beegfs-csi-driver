/*
Copyright 2021 NetApp, Inc. All Rights Reserved.
Licensed under the Apache License, Version 2.0.
*/

package beegfs

import (
	"bytes"
	"context"
	"flag"
	"strings"
	"testing"

	beegfsv1 "github.com/netapp/beegfs-csi-driver/operator/api/v1"
	"k8s.io/klog/v2"
)

// This test verifies that connAuth is properly obfuscated when a BeegfsConfig is logged using our logging
// infrastructure. As an added benefit, it should fail fairly obviously if significant changes are made to our logging
// infrastructure (making it difficult to accidentally circumvent the safeguards we have in place).
func TestStripSecretsFromLogs(t *testing.T) {
	// Set up klog to output to buffer.
	flagSet := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(flagSet)
	if err := flagSet.Set("v", "5"); err != nil { // Ensure all log levels output to buffer.
		t.Fatal(err)
	}
	// -logtostderr must be manually set to false or klog skips outputting to buffer.
	if err := flagSet.Set("logtostderr", "false"); err != nil {
		t.Fatal(err)
	}
	if err := flagSet.Set("stderrthreshold", "4"); err != nil { // Don't ever log to stdErr.
		t.Fatal(err)
	}
	buff := new(bytes.Buffer)
	klog.SetOutput(buff)

	// Create a BeegfsConfig with secrets that should not be logged.
	cfg := beegfsv1.NewBeegfsConfig()
	cfg.ConnAuth = "secret"
	cfg.TLSCert = "tlsCert"

	failIfLogIncorrect := func(buff *bytes.Buffer) {
		stringBuff := buff.String()
		if strings.Contains(stringBuff, "secret") {
			t.Fatalf(`expected to not find plaintext secret in "%s"`, stringBuff)
		}
		if !strings.Contains(stringBuff, "******") {
			t.Fatalf(`expected to find ****** in "%s"`, stringBuff)
		}
	}

	// Verify connAuth is secret at all logging levels.
	LogError(context.TODO(), nil, "some ERROR message", "beegfsConfig", cfg)
	failIfLogIncorrect(buff)
	buff.Reset()
	LogDebug(context.TODO(), "some DEBUG message", "beegfsConfig", cfg)
	failIfLogIncorrect(buff)
	buff.Reset()
	LogVerbose(context.TODO(), "some VERBOSE message", "beegfsConfig", cfg)
	failIfLogIncorrect(buff)
	buff.Reset()
}
