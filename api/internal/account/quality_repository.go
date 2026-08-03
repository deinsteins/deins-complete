package account

import (
	"context"
	"time"
)

type QualityEvent struct {
	ID, CompletionID, InstallationID, EventType         string
	ServerRequestID, Language, Framework, ClientVersion string
	Focus, Mode, Source, FeedbackReason                 string
	LatencyMS                                           int
}

type QualitySummary struct {
	Shown, Accepted          int
	Helpful, NotHelpful      int
	AcceptanceRate           float64
	AverageLatencyMS         float64
	P95LatencyMS             float64
	CacheShown, BackendShown int
}

type QualityDimension struct {
	Kind, Value     string
	Shown, Accepted int
	AcceptanceRate  float64
}

type QualityTrend struct {
	Day             string
	Shown, Accepted int
	AcceptanceRate  float64
	P95LatencyMS    float64
}

type QualityFeedbackReason struct {
	Reason string
	Count  int
}

type AdminQuality struct {
	Days          int
	SamplePercent int
	Summary       QualitySummary
	Dimensions    []QualityDimension
	Trend         []QualityTrend
	Feedback      []QualityFeedbackReason
}

func (r *Repository) RecordQualityEvent(ctx context.Context, event QualityEvent) error {
	if event.FeedbackReason == "" {
		event.FeedbackReason = "none"
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO quality_events
		(id,completion_id,installation_id,event_type,server_request_id,language,framework,focus,mode,source,latency_ms,client_version,feedback_reason)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT DO NOTHING`,
		event.ID, event.CompletionID, event.InstallationID, event.EventType, event.ServerRequestID,
		event.Language, event.Framework, event.Focus, event.Mode, event.Source, event.LatencyMS, event.ClientVersion, event.FeedbackReason)
	return err
}

func (r *Repository) DeleteQualityEventsBefore(ctx context.Context, before time.Time) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM quality_events WHERE created_at < $1`, before)
	return err
}

func (r *Repository) AdminQuality(ctx context.Context, days int) (AdminQuality, error) {
	if days < 1 || days > 90 {
		days = 7
	}
	result := AdminQuality{Days: days}
	err := r.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE event_type='shown')::int,
		count(*) FILTER (WHERE event_type='accepted')::int,
		count(*) FILTER (WHERE event_type='helpful')::int,
		count(*) FILTER (WHERE event_type='not-helpful')::int,
		COALESCE(avg(latency_ms) FILTER (WHERE event_type='shown'),0)::float8,
		COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms) FILTER (WHERE event_type='shown'),0)::float8,
		count(*) FILTER (WHERE event_type='shown' AND source='cache')::int,
		count(*) FILTER (WHERE event_type='shown' AND source='backend')::int
		FROM quality_events WHERE created_at >= now() - ($1 * interval '1 day')`, days).Scan(
		&result.Summary.Shown, &result.Summary.Accepted, &result.Summary.Helpful, &result.Summary.NotHelpful, &result.Summary.AverageLatencyMS,
		&result.Summary.P95LatencyMS, &result.Summary.CacheShown, &result.Summary.BackendShown)
	if err != nil {
		return AdminQuality{}, err
	}
	if result.Summary.Shown > 0 {
		result.Summary.AcceptanceRate = float64(result.Summary.Accepted) / float64(result.Summary.Shown)
	}
	rows, err := r.pool.Query(ctx, `WITH dimensions AS (
		SELECT 'language' AS kind,language AS value,event_type FROM quality_events WHERE created_at >= now() - ($1 * interval '1 day')
		UNION ALL SELECT 'framework',framework,event_type FROM quality_events WHERE created_at >= now() - ($1 * interval '1 day')
		UNION ALL SELECT 'focus',focus,event_type FROM quality_events WHERE created_at >= now() - ($1 * interval '1 day')
		UNION ALL SELECT 'extension-version',client_version,event_type FROM quality_events WHERE created_at >= now() - ($1 * interval '1 day'))
		SELECT kind,value,count(*) FILTER (WHERE event_type='shown')::int,count(*) FILTER (WHERE event_type='accepted')::int
		FROM dimensions GROUP BY kind,value ORDER BY kind,count(*) FILTER (WHERE event_type='shown') DESC,value`, days)
	if err != nil {
		return AdminQuality{}, err
	}
	for rows.Next() {
		var item QualityDimension
		if err := rows.Scan(&item.Kind, &item.Value, &item.Shown, &item.Accepted); err != nil {
			return AdminQuality{}, err
		}
		if item.Shown > 0 {
			item.AcceptanceRate = float64(item.Accepted) / float64(item.Shown)
		}
		result.Dimensions = append(result.Dimensions, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return AdminQuality{}, err
	}
	rows.Close()
	trend, err := r.pool.Query(ctx, `SELECT
		to_char(date_trunc('day',created_at AT TIME ZONE 'UTC'),'YYYY-MM-DD'),
		count(*) FILTER (WHERE event_type='shown')::int,
		count(*) FILTER (WHERE event_type='accepted')::int,
		COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms) FILTER (WHERE event_type='shown'),0)::float8
		FROM quality_events WHERE created_at >= now() - ($1 * interval '1 day')
		GROUP BY date_trunc('day',created_at AT TIME ZONE 'UTC') ORDER BY 1`, days)
	if err != nil {
		return AdminQuality{}, err
	}
	defer trend.Close()
	for trend.Next() {
		var item QualityTrend
		if err := trend.Scan(&item.Day, &item.Shown, &item.Accepted, &item.P95LatencyMS); err != nil {
			return AdminQuality{}, err
		}
		if item.Shown > 0 {
			item.AcceptanceRate = float64(item.Accepted) / float64(item.Shown)
		}
		result.Trend = append(result.Trend, item)
	}
	if err := trend.Err(); err != nil {
		return AdminQuality{}, err
	}
	feedback, err := r.pool.Query(ctx, `SELECT feedback_reason,count(*)::int FROM quality_events
		WHERE created_at >= now() - ($1 * interval '1 day') AND event_type='not-helpful'
		GROUP BY feedback_reason ORDER BY count(*) DESC,feedback_reason`, days)
	if err != nil {
		return AdminQuality{}, err
	}
	defer feedback.Close()
	for feedback.Next() {
		var item QualityFeedbackReason
		if err := feedback.Scan(&item.Reason, &item.Count); err != nil {
			return AdminQuality{}, err
		}
		result.Feedback = append(result.Feedback, item)
	}
	return result, feedback.Err()
}
