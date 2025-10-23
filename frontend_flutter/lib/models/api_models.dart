class SearchResponse {
  final List<SearchResult> data;
  final int total;

  SearchResponse({required this.data, required this.total});

  factory SearchResponse.fromJson(Map<String, dynamic> json) {
    final list = json['results'] as List?;
    return SearchResponse(
      data: list?.map((e) => SearchResult.fromJson(e)).toList() ?? [],
      total: (json['total'] as int?) ?? 0,
    );
  }
}

class SearchResult {
  final int id;
  final int userId;
  final String url;
  final String thumbUrl;

  SearchResult({
    required this.id,
    required this.userId,
    required this.url,
    required this.thumbUrl,
  });

  factory SearchResult.fromJson(Map<String, dynamic> json) {
    return SearchResult(
      id: (json['id'] as int?) ?? 0,
      userId: (json['user_id'] as int?) ?? 0,
      url: json['url'] ?? '',
      thumbUrl: json['thumb_url'] ?? '',
    );
  }
}

class UploadResponse {
  final List<UploadResult> results;
  final UploadSummary summary;

  UploadResponse({
    required this.results,
    required this.summary,
  });

  factory UploadResponse.fromJson(Map<String, dynamic> json) {
    return UploadResponse(
      results: (json['results'] as List)
          .map((item) => UploadResult.fromJson(item))
          .toList(),
      summary: UploadSummary.fromJson(json['summary']),
    );
  }
}

class UploadResult {
  final String filename;
  final String status;
  final MediaInfo? media;
  final String? error; // Error message if status is "failed"

  UploadResult({
    required this.filename,
    required this.status,
    this.media,
    this.error,
  });

  factory UploadResult.fromJson(Map<String, dynamic> json) {
    return UploadResult(
      filename: json['filename'],
      status: json['status'],
      media: json['media'] != null ? MediaInfo.fromJson(json['media']) : null,
      error: json['error'], // Can be null
    );
  }
}

class MediaInfo {
  final int id;
  final String url;
  final String thumbUrl;
  final String mimeType;
  final int sizeBytes;
  final String createdAt;
  final String? takenAt; // Can be null if not available
  final int width;
  final int height;
  final String checksum;

  MediaInfo({
    required this.id,
    required this.url,
    required this.thumbUrl,
    required this.mimeType,
    required this.sizeBytes,
    required this.createdAt,
    this.takenAt, // Optional field
    required this.width,
    required this.height,
    required this.checksum,
  });

  factory MediaInfo.fromJson(Map<String, dynamic> json) {
    return MediaInfo(
      id: json['id'] ?? json['ID'],
      url: json['url'] ?? json['URL'],
      thumbUrl: json['thumb_url'] ?? json['ThumbURL'],
      mimeType: json['mime_type'] ?? json['MimeType'],
      sizeBytes: json['size_bytes'] ?? json['SizeBytes'],
      createdAt: json['created_at'] ?? json['CreatedAt'],
      takenAt: json['taken_at'] ?? json['TakenAt'], // Can be null
      width: json['width'] ?? json['Metadata']?['Width'],
      height: json['height'] ?? json['Metadata']?['Height'],
      checksum: json['checksum'] ?? json['Checksum'],
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
      total: json['total'],
      saved: json['saved'],
      duplicates: json['duplicates'],
      failed: json['failed'],
    );
  }
}
