package paymentpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	transaction "komecore/internal/infra/transactor"
	"komecore/internal/modules/payment/paymentdomain"
	appclock "komecore/pkg/clock"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PaymentChannelDataRepository struct{}

func NewPaymentChannelDataRepository() *PaymentChannelDataRepository {
	return &PaymentChannelDataRepository{}
}

func (r *PaymentChannelDataRepository) Save(
	ctx context.Context,
	exec transaction.Executor,
	data paymentdomain.PaymentChannelData,
) error {
	metadataJSON := "{}"
	if len(data.Metadata) > 0 {
		if b, err := json.Marshal(data.Metadata); err == nil {
			metadataJSON = string(b)
		}
	}

	// Ensure ActionURL has a fallback populated if empty
	actionURL := data.ActionURL
	if actionURL == nil {
		if data.AccountNumber != nil {
			actionURL = data.AccountNumber
		} else if data.QRString != nil {
			actionURL = data.QRString
		} else if data.RedirectURL != nil {
			actionURL = data.RedirectURL
		}
	}

	query := `
		INSERT INTO payment_channel_data (
			id,
			payment_id,
			channel_type,
			display_name,
			account_number,
			qr_string,
			redirect_url,
			action_url,
			metadata,
			expires_at,
			created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11)
		ON CONFLICT (payment_id)
		DO UPDATE SET
			channel_type   = EXCLUDED.channel_type,
			display_name   = EXCLUDED.display_name,
			account_number = EXCLUDED.account_number,
			qr_string      = EXCLUDED.qr_string,
			redirect_url   = EXCLUDED.redirect_url,
			action_url     = EXCLUDED.action_url,
			metadata       = EXCLUDED.metadata,
			expires_at     = EXCLUDED.expires_at
	`

	_, err := exec.Exec(ctx, query,
		data.ID,
		data.PaymentID,
		data.ChannelType,
		data.DisplayName,
		data.AccountNumber,
		data.QRString,
		data.RedirectURL,
		actionURL,
		metadataJSON,
		data.ExpiresAt,
		data.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("save payment channel data failed: %w", err)
	}

	return nil
}

func (r *PaymentChannelDataRepository) GetByPaymentID(
	ctx context.Context,
	exec transaction.Executor,
	paymentID uuid.UUID,
) (*paymentdomain.PaymentChannelData, error) {
	query := `
		SELECT
			id,
			payment_id,
			channel_type,
			display_name,
			account_number,
			qr_string,
			redirect_url,
			action_url,
			metadata,
			expires_at,
			created_at
		FROM payment_channel_data
		WHERE
			payment_id = $1
		LIMIT 1
	`

	var d paymentdomain.PaymentChannelData
	var metadataBytes []byte
	err := exec.QueryRow(ctx, query, paymentID).Scan(
		&d.ID,
		&d.PaymentID,
		&d.ChannelType,
		&d.DisplayName,
		&d.AccountNumber,
		&d.QRString,
		&d.RedirectURL,
		&d.ActionURL,
		&metadataBytes,
		&d.ExpiresAt,
		&d.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query payment channel data by payment_id failed: %w", err)
	}

	populateChannelFallbacks(&d, metadataBytes)

	return &d, nil
}

func (r *PaymentChannelDataRepository) ListByPaymentIDs(
	ctx context.Context,
	exec transaction.Executor,
	paymentIDs []uuid.UUID,
) (map[uuid.UUID]*paymentdomain.PaymentChannelData, error) {
	if len(paymentIDs) == 0 {
		return map[uuid.UUID]*paymentdomain.PaymentChannelData{}, nil
	}

	query := `
		SELECT
			id,
			payment_id,
			channel_type,
			display_name,
			account_number,
			qr_string,
			redirect_url,
			action_url,
			metadata,
			expires_at,
			created_at
		FROM payment_channel_data
		WHERE
			payment_id = ANY($1::uuid[])
	`

	idStrings := make([]string, len(paymentIDs))
	for i, id := range paymentIDs {
		idStrings[i] = id.String()
	}

	rows, err := exec.Query(ctx, query, idStrings)
	if err != nil {
		return nil, fmt.Errorf("query payment channel data by payment ids failed: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]*paymentdomain.PaymentChannelData)
	for rows.Next() {
		var d paymentdomain.PaymentChannelData
		var metadataBytes []byte
		if err := rows.Scan(
			&d.ID,
			&d.PaymentID,
			&d.ChannelType,
			&d.DisplayName,
			&d.AccountNumber,
			&d.QRString,
			&d.RedirectURL,
			&d.ActionURL,
			&metadataBytes,
			&d.ExpiresAt,
			&d.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan payment channel data failed: %w", err)
		}
		populateChannelFallbacks(&d, metadataBytes)
		copied := d
		result[d.PaymentID] = &copied
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment channel data rows failed: %w", err)
	}

	// Initialise the created_at zero value for rows that somehow lack it
	for k, v := range result {
		if v.CreatedAt.IsZero() {
			v.CreatedAt = appclock.Now()
			result[k] = v
		}
	}

	return result, nil
}

func populateChannelFallbacks(d *paymentdomain.PaymentChannelData, metadataBytes []byte) {
	if len(metadataBytes) > 0 {
		var meta map[string]any
		if err := json.Unmarshal(metadataBytes, &meta); err == nil {
			d.Metadata = meta
		}
	}
	if d.Metadata == nil {
		d.Metadata = make(map[string]any)
	}

	// Fallback from legacy action_url if decoupled fields are nil
	if d.AccountNumber == nil && d.QRString == nil && d.RedirectURL == nil && d.ActionURL != nil {
		switch d.ChannelType {
		case paymentdomain.TypeBankTransfer:
			d.AccountNumber = d.ActionURL
		case paymentdomain.TypeQRCode:
			d.QRString = d.ActionURL
		case paymentdomain.TypeEWallet:
			d.RedirectURL = d.ActionURL
		}
	}

	// Fallback to action_url if action_url is nil
	if d.ActionURL == nil {
		if d.AccountNumber != nil {
			d.ActionURL = d.AccountNumber
		} else if d.QRString != nil {
			d.ActionURL = d.QRString
		} else if d.RedirectURL != nil {
			d.ActionURL = d.RedirectURL
		}
	}
}
