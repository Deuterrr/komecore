IF NOT EXISTS (
    SELECT 1 FROM payment_instructions pi
    JOIN payment_methods pm ON pm.id = pi.payment_method_id
    WHERE pm.code = 'bca_va'
)
BEGIN
    INSERT INTO payment_instructions (id, payment_method_id, content, created_at)
    SELECT NEWID(), pm.id, '### BCA Virtual Account Instructions

1. Open **BCA mobile** or access **KlikBCA** / **BCA ATM**.
2. Select **m-Transfer** > **BCA Virtual Account**.
3. Enter Virtual Account number: `{{va_number}}`.
4. Ensure the total payment is **Rp {{amount}}**.
5. Complete payment before **{{expired_at}}**.', GETUTCDATE()
    FROM payment_methods pm
    WHERE pm.code = 'bca_va';
END;

IF NOT EXISTS (
    SELECT 1 FROM payment_instructions pi
    JOIN payment_methods pm ON pm.id = pi.payment_method_id
    WHERE pm.code = 'mandiri_bill'
)
BEGIN
    INSERT INTO payment_instructions (id, payment_method_id, content, created_at)
    SELECT NEWID(), pm.id, '### Mandiri Bill / VA Instructions

1. Open **Livin'' by Mandiri** or visit a **Mandiri ATM**.
2. Select **Bayar** > **Multi Payment** / **Virtual Account**.
3. Enter Company/Biller Code: `70012` and Bill Key: `{{va_number}}`.
4. Confirm total payment: **Rp {{amount}}**.
5. Settle the transaction before **{{expired_at}}**.', GETUTCDATE()
    FROM payment_methods pm
    WHERE pm.code = 'mandiri_bill';
END;

IF NOT EXISTS (
    SELECT 1 FROM payment_instructions pi
    JOIN payment_methods pm ON pm.id = pi.payment_method_id
    WHERE pm.code = 'gopay'
)
BEGIN
    INSERT INTO payment_instructions (id, payment_method_id, content, created_at)
    SELECT NEWID(), pm.id, '### GoPay Payment Instructions

1. Open your **GoPay** or **Gojek** application.
2. Review the invoice total of **Rp {{amount}}** for invoice `{{invoice_number}}`.
3. Confirm payment and enter your PIN.
4. Please complete payment before **{{expired_at}}**.', GETUTCDATE()
    FROM payment_methods pm
    WHERE pm.code = 'gopay';
END;

IF NOT EXISTS (
    SELECT 1 FROM payment_instructions pi
    JOIN payment_methods pm ON pm.id = pi.payment_method_id
    WHERE pm.code = 'shopeepay'
)
BEGIN
    INSERT INTO payment_instructions (id, payment_method_id, content, created_at)
    SELECT NEWID(), pm.id, '### ShopeePay Payment Instructions

1. Open your **Shopee** application.
2. Verify the order amount of **Rp {{amount}}** for invoice `{{invoice_number}}`.
3. Select **ShopeePay** as payment method and authorize with your PIN.
4. Ensure payment is made before **{{expired_at}}**.', GETUTCDATE()
    FROM payment_methods pm
    WHERE pm.code = 'shopeepay';
END;

IF NOT EXISTS (
    SELECT 1 FROM payment_instructions pi
    JOIN payment_methods pm ON pm.id = pi.payment_method_id
    WHERE pm.code = 'qris'
)
BEGIN
    INSERT INTO payment_instructions (id, payment_method_id, content, created_at)
    SELECT NEWID(), pm.id, '### QRIS Payment Instructions

1. Open any supported banking or e-wallet app (BCA Mobile, Livin'', GoPay, OVO, Dana, LinkAja).
2. Scan the provided **QRIS QR Code**.
3. Verify that the merchant name is **KomeCore Store** and the amount is **Rp {{amount}}**.
4. Confirm payment before **{{expired_at}}**.', GETUTCDATE()
    FROM payment_methods pm
    WHERE pm.code = 'qris';
END;
