CREATE TABLE outbox_events (
    id UNIQUEIDENTIFIER NOT NULL DEFAULT NEWID(),
    event_type NVARCHAR(100) NOT NULL,
    payload NVARCHAR(MAX) NOT NULL,
    status NVARCHAR(20) NOT NULL CONSTRAINT df_outbox_events_status DEFAULT 'pending',
    retry_count INT NOT NULL CONSTRAINT df_outbox_events_retry_count DEFAULT 0,
    max_retries INT NOT NULL CONSTRAINT df_outbox_events_max_retries DEFAULT 5,
    last_error NVARCHAR(MAX),
    scheduled_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    processed_at DATETIMEOFFSET,
    created_at DATETIMEOFFSET NOT NULL DEFAULT GETUTCDATE(),
    CONSTRAINT pk_outbox_events PRIMARY KEY (id),
    CONSTRAINT chk_outbox_events_status CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

CREATE INDEX idx_outbox_events_pending 
    ON outbox_events (scheduled_at) 
    WHERE status = 'pending';
