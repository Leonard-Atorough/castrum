package components

import (
	"reflect"
	"testing"
)

func TestNewAudio(t *testing.T) {
	type args struct {
		trackID  string
		autoplay bool
		volume   float64
	}
	tests := []struct {
		name string
		args args
		want AudioPlayer
	}{
		{
			name: "with autoplay",
			args: args{
				trackID:  "track1",
				autoplay: true,
			},
			want: AudioPlayer{
				TrackID:  "track1",
				Autoplay: true,
				Playing:  false,
			},
		},
		{
			name: "without autoplay",
			args: args{
				trackID:  "track2",
				autoplay: false,
			},
			want: AudioPlayer{
				TrackID:  "track2",
				Autoplay: false,
				Playing:  false,
			},
		},
		{
			name: "with zero volume",
			args: args{
				trackID:  "track3",
				autoplay: true,
			},
			want: AudioPlayer{
				TrackID:  "track3",
				Autoplay: true,
				Playing:  false,
			},
		},
		{
			name: "with max volume",
			args: args{
				trackID:  "track4",
				autoplay: false,
			},
			want: AudioPlayer{
				TrackID:  "track4",
				Autoplay: false,
				Playing:  false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAudio(tt.args.trackID, tt.args.autoplay); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewAudio() = %v, want %v", got, tt.want)
			}
		})
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
			a:       AudioPlayer{TrackID: "track1", Autoplay: true, Playing: false},
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
	tests := []struct {
		name    string
		a       AudioPlayer
		want    map[string]any
		wantErr bool
	}{
		{
			name:    "valid audio",
			a:       AudioPlayer{TrackID: "track1", Autoplay: true, Playing: false},
			want:    map[string]any{"trackID": "track1", "autoplay": true, "playing": false},
			wantErr: false,
		},
		{
			name:    "valid audio without autoplay",
			a:       AudioPlayer{TrackID: "track4", Autoplay: false, Playing: false},
			want:    map[string]any{"trackID": "track4", "autoplay": false, "playing": false},
			wantErr: false,
		},
		{
			name:    "valid audio with autoplay false and volume 1",
			a:       AudioPlayer{TrackID: "track5", Autoplay: false, Playing: false},
			want:    map[string]any{"trackID": "track5", "autoplay": false, "playing": false},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.a.Serialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("Audio.Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Audio.Serialize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAudio_Deserialize(t *testing.T) {
	type args struct {
		data map[string]any
	}
	tests := []struct {
		name    string
		a       AudioPlayer
		args    args
		want    AudioPlayer
		wantErr bool
	}{
		{
			name:    "valid audio",
			a:       AudioPlayer{},
			args:    args{data: map[string]any{"trackID": "track1", "autoplay": true, "playing": false}},
			want:    AudioPlayer{TrackID: "track1", Autoplay: true, Playing: false},
			wantErr: false,
		},
		{
			name:    "invalid audio with empty trackID",
			a:       AudioPlayer{},
			args:    args{data: map[string]any{"trackID": "", "autoplay": true, "playing": false}},
			want:    AudioPlayer{TrackID: "", Autoplay: true, Playing: false},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.a.Deserialize(tt.args.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Audio.Deserialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Audio.Deserialize() = %v, want %v", got, tt.want)
			}
		})
	}
}
