import 'dart:math';

import 'package:frontend_flutter/data/models.dart';

class Relevance {
  /// Keep items whose score is at least (alpha * maxScore).
  /// alpha in [0,1]. Higher alpha = stricter (fewer results).
  static List<MediaResponse> relativeToMax(List<MediaResponse> items, {double alpha = 0.85}) {
    final withScore = items.where((e) => e.score != null).toList();
    if (withScore.isEmpty) return const [];

    withScore.sort((a, b) => b.score!.compareTo(a.score!)); // high → low
    final maxScore = withScore.first.score!;
    final threshold = maxScore * alpha;

    return withScore.where((e) => e.score! >= threshold).toList();
  }

  /// Statistical outlier filtering: keep results above mean + k*stddev.
  /// Same approach used by category search. Works much better with CLIP
  /// scores that cluster tightly.
  /// [k] controls strictness: 1.0 = more results, 1.5 = balanced, 2.0 = strict.
  static List<MediaResponse> statisticalOutliers(List<MediaResponse> items, {double k = 1.0}) {
    final withScore = items.where((e) => e.score != null).toList();
    if (withScore.isEmpty) return const [];
    if (withScore.length == 1) return withScore;

    withScore.sort((a, b) => b.score!.compareTo(a.score!));

    final scores = withScore.map((r) => r.score!).toList();
    final mean = scores.reduce((a, b) => a + b) / scores.length;
    final variance =
        scores.map((s) => (s - mean) * (s - mean)).reduce((a, b) => a + b) /
            scores.length;
    final stddev = sqrt(variance);
    final cutoff = mean + k * stddev;

    final filtered = withScore.where((e) => e.score! >= cutoff).toList();
    // Always return at least the top result
    if (filtered.isEmpty) return [withScore.first];
    return filtered;
  }
}
