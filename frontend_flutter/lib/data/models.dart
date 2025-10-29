class MediaResponse {
  final int id;
  final int userId;
  final String url;
  final String thumbUrl;
  final double? score; // optional

  MediaResponse({
    required this.id,
    required this.userId,
    required this.url,
    required this.thumbUrl,
    this.score,
  });

  factory MediaResponse.fromJson(Map<String, dynamic> json) {
    return MediaResponse(
      id: (json['id'] as num).toInt(),
      userId: (json['user_id'] as num).toInt(),
      url: json['url'] as String,
      thumbUrl: json['thumb_url'] as String,
      score: json['score'] == null ? null : (json['score'] as num).toDouble(),
    );
  }
}

class SearchResponse {
  final List<MediaResponse> results;
  final int total;

  SearchResponse({required this.results, required this.total});

  factory SearchResponse.fromJson(Map<String, dynamic> json) {
    final list = (json['results'] as List? ?? []);
    return SearchResponse(
      results: list
          .map((e) => MediaResponse.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: (json['total'] as num? ?? 0).toInt(),
    );
  }
}

class UploadItem {
  final int id;
  final String url;
  final String thumbUrl;
  final String mimeType;
  final int sizeBytes;
  final int width;
  final int height;
  final String checksum;
  final String? takenAt;   // RFC3339 or null
  final String createdAt;  // RFC3339

  UploadItem({
    required this.id,
    required this.url,
    required this.thumbUrl,
    required this.mimeType,
    required this.sizeBytes,
    required this.width,
    required this.height,
    required this.checksum,
    required this.takenAt,
    required this.createdAt,
  });

  factory UploadItem.fromJson(Map<String, dynamic> json) {
    final t = json['taken_at'] as String?;
    return UploadItem(
      id: (json['id'] as num).toInt(),
      url: json['url'] as String,
      thumbUrl: json['thumb_url'] as String,
      mimeType: json['mime_type'] as String,
      sizeBytes: (json['size_bytes'] as num).toInt(),
      width: (json['width'] as num).toInt(),
      height: (json['height'] as num).toInt(),
      checksum: json['checksum'] as String,
      takenAt: (t == null || t.isEmpty) ? null : t,
      createdAt: json['created_at'] as String,
    );
  }
}

class UploadResult {
  final String filename;
  final String status;
  final String? error;
  final UploadItem? media;

  UploadResult({
    required this.filename,
    required this.status,
    this.error,
    this.media,
  });

  factory UploadResult.fromJson(Map<String, dynamic> json) {
    return UploadResult(
      filename: json['filename'] as String,
      status: json['status'] as String,
      error: json['error'] as String?,
      media: json['media'] == null
          ? null
          : UploadItem.fromJson(json['media'] as Map<String, dynamic>),
    );
  }
}

class UploadSummary {
  final int total;
  final int saved;
  final int duplicates;
  final int failed;

  UploadSummary({
    required this.total,
    required this.saved,
    required this.duplicates,
    required this.failed,
  });

  factory UploadSummary.fromJson(Map<String, dynamic> json) {
    return UploadSummary(
      total: (json['total'] as num).toInt(),
      saved: (json['saved'] as num).toInt(),
      duplicates: (json['duplicates'] as num).toInt(),
      failed: (json['failed'] as num).toInt(),
    );
  }
}

class UploadResponseModel {
  final List<UploadResult> results;
  final UploadSummary summary;

  UploadResponseModel({required this.results, required this.summary});

  factory UploadResponseModel.fromJson(Map<String, dynamic> json) {
    final list = (json['results'] as List? ?? []);
    return UploadResponseModel(
      results: list
          .map((e) => UploadResult.fromJson(e as Map<String, dynamic>))
          .toList(),
      summary: UploadSummary.fromJson(json['summary'] as Map<String, dynamic>),
    );
  }
}
