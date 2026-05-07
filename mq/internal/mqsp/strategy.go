package mqsp

import (
	"context"
	"time"

	"github.com/MrMiaoMIMI/goshared/mq/mqspi"
)

type consumerFailurePolicyStrategy struct {
	policy *mqspi.ConsumerFailurePolicy
}

func newConsumerFailurePolicyStrategy(policy *mqspi.ConsumerFailurePolicy) mqspi.ConsumerFailureStrategy {
	return &consumerFailurePolicyStrategy{policy: policy}
}

func (s *consumerFailurePolicyStrategy) Decide(ctx context.Context, failure *mqspi.ConsumerFailure) mqspi.ConsumerFailureDecision {
	policy := normalizeConsumerFailurePolicy(s.policy)
	if policy.Handler != nil {
		policy.Handler(ctx, failure)
	}

	if policy.MaxAttempts > 0 && failure != nil && failure.Attempt >= policy.MaxAttempts {
		if policy.FinalAction == mqspi.ConsumerFailureActionRetry {
			return mqspi.ConsumerFailureDecision{
				Action:  mqspi.ConsumerFailureActionRetry,
				Backoff: consumerFailureBackoff(policy, failureAttempt(failure)),
			}
		}
		return mqspi.ConsumerFailureDecision{Action: policy.FinalAction}
	}

	return mqspi.ConsumerFailureDecision{
		Action:  mqspi.ConsumerFailureActionRetry,
		Backoff: consumerFailureBackoff(policy, failureAttempt(failure)),
	}
}

func normalizeConsumerFailurePolicy(policy *mqspi.ConsumerFailurePolicy) mqspi.ConsumerFailurePolicy {
	if policy == nil {
		return mqspi.ConsumerFailurePolicy{
			InitialBackoff:    mqspi.DefaultFailureInitialBackoff,
			MaxBackoff:        mqspi.DefaultFailureMaxBackoff,
			BackoffMultiplier: mqspi.DefaultFailureBackoffMultiple,
			FinalAction:       mqspi.ConsumerFailureActionStop,
		}
	}

	out := *policy
	if out.InitialBackoff <= 0 {
		out.InitialBackoff = mqspi.DefaultFailureInitialBackoff
	}
	if out.MaxBackoff <= 0 {
		out.MaxBackoff = mqspi.DefaultFailureMaxBackoff
	}
	if out.BackoffMultiplier <= 1 {
		out.BackoffMultiplier = mqspi.DefaultFailureBackoffMultiple
	}
	if out.FinalAction == "" {
		out.FinalAction = mqspi.ConsumerFailureActionStop
	}
	return out
}

func consumerFailureBackoff(policy mqspi.ConsumerFailurePolicy, attempt int) time.Duration {
	if attempt <= 1 {
		return policy.InitialBackoff
	}

	backoff := policy.InitialBackoff
	for i := 1; i < attempt; i++ {
		next := time.Duration(float64(backoff) * policy.BackoffMultiplier)
		if next <= 0 || next >= policy.MaxBackoff {
			return policy.MaxBackoff
		}
		backoff = next
	}
	return backoff
}

func failureAttempt(failure *mqspi.ConsumerFailure) int {
	if failure == nil || failure.Attempt <= 0 {
		return 1
	}
	return failure.Attempt
}
