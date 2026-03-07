package utils

import "testing"

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "合法密码",
			password: "Abcdef1!",
			wantErr:  false,
		},
		{
			name:     "长度不足",
			password: "Ab1!",
			wantErr:  true,
		},
		{
			name:     "缺少小写字母",
			password: "ABCDEF1!",
			wantErr:  true,
		},
		{
			name:     "缺少大写字母",
			password: "abcdef1!",
			wantErr:  true,
		},
		{
			name:     "缺少数字",
			password: "Abcdefg!",
			wantErr:  true,
		},
		{
			name:     "缺少特殊字符",
			password: "Abcdefg1",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePassword(tt.password)
			if tt.wantErr && err == nil {
				t.Fatalf("期望错误，但返回 nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("期望通过，但返回错误: %v", err)
			}
		})
	}
}

