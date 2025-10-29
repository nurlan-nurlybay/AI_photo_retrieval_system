import 'models.dart';

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
}
