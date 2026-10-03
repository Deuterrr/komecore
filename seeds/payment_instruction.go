package seeds

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type defaultInstruction struct {
	MethodCode string
	Content    string
}

var defaultInstructions = []defaultInstruction{
	{
		MethodCode: "bca_va",
		Content: "### BCA Virtual Account Instructions\n\n" +
			"1. Open **BCA mobile** or access **KlikBCA** / **BCA ATM**.\n" +
			"2. Select **m-Transfer** > **BCA Virtual Account**.\n" +
			"3. Enter Virtual Account number: `{{va_number}}`.\n" +
			"4. Ensure the total payment is **Rp {{amount}}**.\n" +
			"5. Complete payment before **{{expired_at}}**.",
	},
	{
		MethodCode: "mandiri_bill",
		Content: "### Mandiri Bill / VA Instructions\n\n" +
			"1. Open **Livin' by Mandiri** or visit a **Mandiri ATM**.\n" +
			"2. Select **Bayar** > **Multi Payment** / **Virtual Account**.\n" +
			"3. Enter Company/Biller Code: `70012` and Bill Key: `{{va_number}}`.\n" +
			"4. Confirm total payment: **Rp {{amount}}**.\n" +
			"5. Settle the transaction before **{{expired_at}}**.",
	},
	{
		MethodCode: "gopay",
		Content: "### GoPay Payment Instructions\n\n" +
			"1. Open your **GoPay** or **Gojek** application.\n" +
			"2. Review the invoice total of **Rp {{amount}}** for invoice `{{invoice_number}}`.\n" +
			"3. Confirm payment and enter your PIN.\n" +
			"4. Please complete payment before **{{expired_at}}**.",
	},
	{
		MethodCode: "shopeepay",
		Content: "### ShopeePay Payment Instructions\n\n" +
			"1. Open your **Shopee** application.\n" +
			"2. Verify the order amount of **Rp {{amount}}** for invoice `{{invoice_number}}`.\n" +
			"3. Select **ShopeePay** as payment method and authorize with your PIN.\n" +
			"4. Ensure payment is made before **{{expired_at}}**.",
	},
	{
		MethodCode: "qris",
		Content: "### QRIS Payment Instructions\n\n" +
			"1. Open any supported banking or e-wallet app (BCA Mobile, Livin', GoPay, OVO, Dana, LinkAja).\n" +
			"2. Scan the provided **QRIS QR Code**.\n" +
			"3. Verify that the merchant name is **KomeCore Store** and the amount is **Rp {{amount}}**.\n" +
			"4. Confirm payment before **{{expired_at}}**.",
	},
}

func SeedPaymentInstructions(ctx context.Context, pool *pgxpool.Pool) error {
	seeded, err := paymentInstructionAlreadySeeded(ctx, pool)
	if err != nil {
		return err
	}

	if seeded {
		log.Println("database: payment instructions already seeded, skipping")
		return nil
	}

	log.Println("database: seeding payment instructions")

	query := `
		INSERT INTO payment_instructions (id, payment_method_id, content, created_at)
		SELECT gen_random_uuid(), pm.id, $1, NOW()
		FROM payment_methods pm
		WHERE pm.code = $2
		ON CONFLICT (payment_method_id) DO NOTHING
	`

	for _, item := range defaultInstructions {
		if _, err := pool.Exec(ctx, query, item.Content, item.MethodCode); err != nil {
			return fmt.Errorf("failed to seed payment instruction for %s: %w", item.MethodCode, err)
		}
	}

	if err := markPaymentInstructionSeeded(ctx, pool); err != nil {
		return fmt.Errorf("failed to mark payment instruction seed: %w", err)
	}

	return nil
}

func paymentInstructionAlreadySeeded(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var exists bool

	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM seed_versions WHERE name = $1
		)
	`, "payment_instruction_v1").Scan(&exists)

	return exists, err
}

func markPaymentInstructionSeeded(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO seed_versions (name, version)
		VALUES ($1, $2)
		ON CONFLICT (name) DO NOTHING
	`, "payment_instruction_v1", "1.0")

	return err
}
