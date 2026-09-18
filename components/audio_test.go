package components

import (
	"reflect"
	"testing"
)

func TestNewAudio(t *testing.T) {
	tests := []struct {
		name     string
		trackID  string
		volume   float64
		mode     PlaybackMode
		autoplay bool
		want     AudioPlayer
	}{
		{
			name:     "autoplay with persist mode",
			trackID:  "track1",
			volume:   0.8,
			mode:     PlaybackPersist,
			autoplay: true,
			want: AudioPlayer{
				TrackID:  "track1",
				Volume:   0.8,
				Mode:     PlaybackPersist,
				Playing:  true,
				Autoplay: true,
			},
		},
		{
			name:     "no autoplay with despawn mode",
			trackID:  "track2",
			volume:   1.0,
			mode:     PlaybackDespawn,
			autoplay: false,
			want: AudioPlayer{
				TrackID:  "track2",
				Volume:   1.0,
				Mode:     PlaybackDespawn,
				Playing:  false,
				Autoplay: false,
			},
		},
		{
			name:     "zero volume",
			trackID:  "track3",
			volume:   0,
			mode:     PlaybackDespawn,
			autoplay: true,
			want: AudioPlayer{
				TrackID:  "track3",
				Volume:   0,
				Mode:     PlaybackDespawn,
				Playing:  true,
				Autoplay: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAudio(tt.trackID, tt.volume, tt.mode, tt.autoplay); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAudio() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewAudioClampsVolume(t *testing.T) {
	track := NewAudio("sfx", -0.5, PlaybackDespawn, true)
	if track.Volume != 0 {
		t.Errorf("Volume = %v, want 0 (clamped from -0.5)", track.Volume)
	}

	track = NewAudio("sfx", 2.0, PlaybackDespawn, true)
	if track.Volume != 1 {
		t.Errorf("Volume = %v, want 1 (clamped from 2.0)", track.Volume)
	}
}

func TestPlaybackModeConstantValues(t *testing.T) {
	if PlaybackPersist != 0 {
		t.Errorf("PlaybackPersist = %d, want 0", PlaybackPersist)
	}
	if PlaybackDespawn != 1 {
		t.Errorf("PlaybackDespawn = %d, want 1", PlaybackDespawn)
	}
}

func TestAudio_Validate(t *testing.T) {
	tests := []struct {
		name    string
		a       AudioPlayer
		wantErr bool
	}{
		{
			name:    "valid audio",
			a:       AudioPlayer{TrackID: "track1", Volume: 0.8, Mode: PlaybackPersist, Autoplay: true, Playing: false},
			wantErr: false,
		},
		{
			name:    "invalid audio with empty trackID",
			a:       AudioPlayer{TrackID: "", Autoplay: true, Playing: false},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.a.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Audio.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAudio_Serialize(t *testing.T) {
	a := AudioPlayer{
		TrackID:  "track1",
		Volume:   0.75,
		Mode:     PlaybackDespawn,
		Playing:  false,
		Autoplay: true,
	}
	want := map[string]any{
		"trackID":  "track1",
		"volume":   0.75,
		"mode":     float64(PlaybackDespawn),
		"playing":  false,
		"autoplay": true,
	}

	got, err := a.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Serialize() = %v, want %v", got, want)
	}
}

func TestAudio_Deserialize(t *testing.T) {
	tests := []struct {
		name    string
		data    map[string]any
		want    AudioPlayer
		wantErr bool
	}{
		{
			name: "all fields",
			data: map[string]any{
				"trackID":  "track1",
				"volume":   0.75,
				"mode":     float64(PlaybackDespawn),
				"playing":  false,
				"autoplay": true,
			},
			want: AudioPlayer{
				TrackID:  "track1",
				Volume:   0.75,
				Mode:     PlaybackDespawn,
				Playing:  false,
				Autoplay: true,
			},
			wantErr: false,
		},
		{
			name: "missing fields default to zero",
			data: map[string]any{
				"trackID": "track2",
			},
			want: AudioPlayer{
				TrackID: "track2",
			},
			wantErr: false,
		},
		{
			name: "empty trackID errors",
			data: map[string]any{
				"trackID": "",
			},
			want:    AudioPlayer{TrackID: ""},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AudioPlayer{}.Deserialize(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Deserialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Deserialize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAudio_SerializeDeserializeRoundTrip(t *testing.T) {
	original := AudioPlayer{
		TrackID:  "bgm",
		Volume:   0.5,
		Mode:     PlaybackPersist,
		Playing:  true,
		Autoplay: true,
	}
	data, err := original.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	restored, err := AudioPlayer{}.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if !reflect.DeepEqual(restored, original) {
		t.Errorf("round-trip mismatch: got %v, want %v", restored, original)
	}
}
