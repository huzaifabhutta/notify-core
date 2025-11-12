package channels_test

import (
	"testing"

	"github.com/huzaifabhutta/notify-core/internal/channels"
	_ "github.com/huzaifabhutta/notify-core/internal/channels/email"
	_ "github.com/huzaifabhutta/notify-core/internal/channels/sms"
	_ "github.com/huzaifabhutta/notify-core/internal/channels/whatsapp"
)

func TestChannelRegistration_AllChannels(t *testing.T) {
	// Test that all channels are auto-registered on import
	expectedChannels := []struct {
		name         string
		channelType  channels.Type
		minAdapters  int
		minFeatures  int
	}{
		{
			name:         "email",
			channelType:  channels.TypeMessaging,
			minAdapters:  2, // smtp, ses
			minFeatures:  3, // attachments, rich_text, templates
		},
		{
			name:         "sms",
			channelType:  channels.TypeMessaging,
			minAdapters:  1, // sns
			minFeatures:  1, // At least templates
		},
		{
			name:         "whatsapp",
			channelType:  channels.TypeMessaging,
			minAdapters:  1, // whatsapp-cloud
			minFeatures:  4, // attachments, rich_text, templates, delivery_receipt
		},
	}

	for _, tc := range expectedChannels {
		t.Run(tc.name, func(t *testing.T) {
			// Check if channel is registered
			if !channels.Has(tc.name) {
				t.Fatalf("Expected channel %s to be registered", tc.name)
			}

			// Get metadata
			meta := channels.GetMetadata(tc.name)
			if meta == nil {
				t.Fatalf("Expected metadata for channel %s", tc.name)
			}

			// Verify metadata fields
			if meta.Name != tc.name {
				t.Errorf("Expected channel name %s, got %s", tc.name, meta.Name)
			}

			if meta.Type != tc.channelType {
				t.Errorf("Expected channel type %s, got %s", tc.channelType, meta.Type)
			}

			if meta.Version == "" {
				t.Error("Expected non-empty version")
			}

			if meta.Description == "" {
				t.Error("Expected non-empty description")
			}

			// Verify adapters
			if len(meta.Adapters) < tc.minAdapters {
				t.Errorf("Expected at least %d adapters, got %d", tc.minAdapters, len(meta.Adapters))
			}

			// Verify features
			if len(meta.Features) < tc.minFeatures {
				t.Errorf("Expected at least %d features, got %d", tc.minFeatures, len(meta.Features))
			}

			t.Logf("Channel %s registered successfully with %d adapters and %d features",
				tc.name, len(meta.Adapters), len(meta.Features))
		})
	}
}

func TestChannelList_AllChannels(t *testing.T) {
	// Get all registered channels
	allChannels := channels.List()

	// Should have at least 3 channels (email, sms, whatsapp)
	if len(allChannels) < 3 {
		t.Errorf("Expected at least 3 channels, got %d", len(allChannels))
	}

	// Verify specific channels exist
	expectedChannels := []string{"email", "sms", "whatsapp"}
	for _, expected := range expectedChannels {
		found := false
		for _, channel := range allChannels {
			if channel == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find channel %s in list", expected)
		}
	}

	t.Logf("Total channels registered: %d", len(allChannels))
	t.Logf("Channels: %v", allChannels)
}

func TestChannelListByType_Messaging(t *testing.T) {
	// Get all messaging channels
	messagingChannels := channels.ListByType(channels.TypeMessaging)

	// Should have at least 3 messaging channels
	if len(messagingChannels) < 3 {
		t.Errorf("Expected at least 3 messaging channels, got %d", len(messagingChannels))
	}

	// All should be messaging type
	for _, channelName := range messagingChannels {
		meta := channels.GetMetadata(channelName)
		if meta == nil {
			t.Fatalf("Expected metadata for channel %s", channelName)
		}

		if meta.Type != channels.TypeMessaging {
			t.Errorf("Expected channel %s to be messaging type, got %s", channelName, meta.Type)
		}
	}

	t.Logf("Messaging channels: %v", messagingChannels)
}

func TestChannelFeatures(t *testing.T) {
	tests := []struct {
		channel     string
		feature     channels.Feature
		shouldSupport bool
	}{
		// Email features
		{"email", channels.FeatureAttachments, true},
		{"email", channels.FeatureRichText, true},
		{"email", channels.FeatureTemplates, true},

		// SMS features
		{"sms", channels.FeatureAttachments, false}, // SMS doesn't support attachments
		{"sms", channels.FeatureRichText, false},    // SMS is plain text
		{"sms", channels.FeatureTemplates, true},

		// WhatsApp features
		{"whatsapp", channels.FeatureAttachments, true},
		{"whatsapp", channels.FeatureRichText, true},
		{"whatsapp", channels.FeatureTemplates, true},
		{"whatsapp", channels.FeatureDeliveryReceipt, true},
	}

	for _, tt := range tests {
		t.Run(tt.channel+"_"+string(tt.feature), func(t *testing.T) {
			meta := channels.GetMetadata(tt.channel)
			if meta == nil {
				t.Fatalf("Channel %s not found", tt.channel)
			}

			// Check if feature is in metadata
			hasFeature := false
			for _, f := range meta.Features {
				if f == tt.feature {
					hasFeature = true
					break
				}
			}

			if hasFeature != tt.shouldSupport {
				t.Errorf("Channel %s feature %s: expected %v, got %v",
					tt.channel, tt.feature, tt.shouldSupport, hasFeature)
			}
		})
	}
}

func TestChannelAdapters(t *testing.T) {
	tests := []struct {
		channel         string
		expectedAdapters []string
	}{
		{
			channel:         "email",
			expectedAdapters: []string{"smtp", "ses"},
		},
		{
			channel:         "sms",
			expectedAdapters: []string{"sns"},
		},
		{
			channel:         "whatsapp",
			expectedAdapters: []string{"whatsapp-cloud"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			meta := channels.GetMetadata(tt.channel)
			if meta == nil {
				t.Fatalf("Channel %s not found", tt.channel)
			}

			// Verify all expected adapters are present
			for _, expectedAdapter := range tt.expectedAdapters {
				found := false
				for _, adapter := range meta.Adapters {
					if adapter == expectedAdapter {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected adapter %s for channel %s", expectedAdapter, tt.channel)
				}
			}

			t.Logf("Channel %s adapters: %v", tt.channel, meta.Adapters)
		})
	}
}

func BenchmarkChannelRegistryLookup(b *testing.B) {
	// Benchmark channel lookup performance
	channelNames := []string{"email", "sms", "whatsapp"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		channelName := channelNames[i%len(channelNames)]
		_ = channels.GetMetadata(channelName)
	}
}

func BenchmarkChannelList(b *testing.B) {
	// Benchmark listing all channels
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = channels.List()
	}
}
