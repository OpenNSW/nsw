package gateways

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapGovPayStatus(t *testing.T) {
	cases := map[string]struct {
		in      string
		want    WebhookStatus
		wantErr bool
	}{
		"paid":      {in: "paid", want: WebhookStatusSuccess},
		"completed": {in: "COMPLETED", want: WebhookStatusSuccess},
		"success":   {in: "success", want: WebhookStatusSuccess},
		"declined":  {in: "declined", want: WebhookStatusFailed},
		"rejected":  {in: "REJECTED", want: WebhookStatusFailed},
		"pending":   {in: " Pending ", want: WebhookStatusPending}, // trims + case-insensitive
		"unknown":   {in: "weird", wantErr: true},
		"empty":     {in: "", wantErr: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := mapGovPayStatus(tc.in)
			if tc.wantErr {
				require.ErrorIs(t, err, ErrUnsupportedWebhookStatus)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGovPay_ParseWebhook_NormalizesStatus(t *testing.T) {
	g := &GovPayGateway{}
	body := []byte(`{"reference_number":"TNSW1","status":"paid","amount":"1500.00","currency":"LKR","payment_method":"CC"}`)

	p, err := g.ParseWebhook(context.Background(), body, nil)
	require.NoError(t, err)
	assert.Equal(t, "TNSW1", p.ReferenceNumber)
	assert.Equal(t, WebhookStatusSuccess, p.Status)
	assert.Equal(t, "CC", p.PaymentMethod)
	assert.True(t, p.Amount.Equal(decimal.RequireFromString("1500.00")))
}

func TestGovPay_ParseWebhook_UnknownStatus(t *testing.T) {
	g := &GovPayGateway{}
	_, err := g.ParseWebhook(context.Background(), []byte(`{"status":"weird"}`), nil)
	require.ErrorIs(t, err, ErrUnsupportedWebhookStatus)
}

func TestGovPay_ParseWebhook_InvalidJSON(t *testing.T) {
	g := &GovPayGateway{}
	_, err := g.ParseWebhook(context.Background(), []byte(`not json`), nil)
	require.Error(t, err)
}

func TestGovPay_HandleValidateReference(t *testing.T) {
	g := &GovPayGateway{}
	reqData := []byte(`{"transactionId":"abc","subinstId":"s1","serviceId":"sv1","serviceName":"App Fee"}`)

	t.Run("payable", func(t *testing.T) {
		resp, err := g.HandleValidateReference(context.Background(), &ValidationTransaction{ReferenceNumber: "abc"}, true, reqData)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.HTTPStatus)

		var out GovPayValidateResponse
		require.NoError(t, json.Unmarshal(resp.Payload, &out))
		assert.Equal(t, "Success", out.Message)
		assert.Equal(t, "abc", out.TransactionID)
	})

	t.Run("not payable", func(t *testing.T) {
		resp, err := g.HandleValidateReference(context.Background(), nil, false, reqData)
		require.NoError(t, err)

		var out GovPayValidateResponse
		require.NoError(t, json.Unmarshal(resp.Payload, &out))
		assert.NotEqual(t, "Success", out.Message)
		assert.Equal(t, "abc", out.TransactionID) // still echoes the request fields
	})
}

func TestGovPay_ExtractReferenceNumber(t *testing.T) {
	g := &GovPayGateway{}

	ref, err := g.ExtractReferenceNumber(context.Background(), []byte(`{"transactionId":"abc"}`))
	require.NoError(t, err)
	assert.Equal(t, "abc", ref)

	_, err = g.ExtractReferenceNumber(context.Background(), []byte(`{}`))
	require.Error(t, err)
}

func TestGovPay_CreateSession(t *testing.T) {
	g := &GovPayGateway{}
	resp, err := g.CreateSession(context.Background(), SessionRequest{})
	require.NoError(t, err)
	assert.Equal(t, FlowTypeInstruction, resp.Type)
	assert.NotEmpty(t, resp.Instructions)
}

func TestNewGovPayGateway(t *testing.T) {
	gw, err := NewGovPayGateway([]byte(`{"BaseURL":"https://sandbox.govpay.lk"}`))
	require.NoError(t, err)
	g, ok := gw.(*GovPayGateway)
	require.True(t, ok)
	assert.Equal(t, "https://sandbox.govpay.lk", g.cfg.BaseURL)

	_, err = NewGovPayGateway([]byte(`not json`))
	require.Error(t, err)
}

func TestGovPay_ExtractReferenceNumber_InvalidJSON(t *testing.T) {
	g := &GovPayGateway{}
	_, err := g.ExtractReferenceNumber(context.Background(), []byte(`not json`))
	require.Error(t, err)
}

func TestGovPay_HandleValidateReference_InvalidJSON(t *testing.T) {
	g := &GovPayGateway{}
	_, err := g.HandleValidateReference(context.Background(), nil, true, []byte(`not json`))
	require.Error(t, err)
}
