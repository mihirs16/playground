# The media CDN: the public read path for uploaded media, served CDN-direct from
# the private S3 media bucket and never through the custodian origin (ADR-0002).
# This is a second, dedicated CloudFront distribution aliased to cdn.<domain> —
# distinct from the custodian/persona edge (edge.tf) because CloudFront routes to
# origins by path, not by Host, so a separate hostname wants its own distribution.
# The media bucket stays fully private: it is reachable only by this distribution
# through Origin Access Control, authorized by a bucket policy scoped to the
# distribution's ARN — not a public policy, so buckets.tf's access block stands.

# cdn.<domain>'s viewer certificate, issued in us-east-1 like the edge's (the only
# region CloudFront reads viewer certificates from) and DNS-validated through the
# zone. AWS holds the private key; no certificate material lands on disk.
resource "aws_acm_certificate" "media_cdn" {
  provider          = aws.us_east_1
  domain_name       = var.cdn_domain_name
  validation_method = "DNS"

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "media_cdn_cert_validation" {
  for_each = {
    for dvo in aws_acm_certificate.media_cdn.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      type   = dvo.resource_record_type
      record = dvo.resource_record_value
    }
  }

  zone_id         = aws_route53_zone.playground.zone_id
  name            = each.value.name
  type            = each.value.type
  ttl             = 60
  records         = [each.value.record]
  allow_overwrite = true
}

resource "aws_acm_certificate_validation" "media_cdn" {
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.media_cdn.arn
  validation_record_fqdns = [for r in aws_route53_record.media_cdn_cert_validation : r.fqdn]
}

# Origin Access Control: CloudFront signs each origin request with SigV4 so the
# private bucket can authorize it, replacing the legacy Origin Access Identity.
resource "aws_cloudfront_origin_access_control" "media" {
  name                              = "custodian-media"
  description                       = "OAC for the media CDN over the private media bucket"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

# Media is immutable-by-key — custodian never overwrites a reserved key — so a
# long TTL is safe and a delete is a bucket-object removal, not an invalidation.
# Every object caches for the same fixed lifetime: min = default = max pins the
# TTL, and since CloudFront clamps any origin Cache-Control into [min, max], an
# equal min and max makes the edge lifetime uniform regardless of what (if
# anything) S3 sends per object. The cache key is the path alone — no cookies,
# headers, or query strings enter it, so one key per object across all viewers.
resource "aws_cloudfront_cache_policy" "media" {
  name        = "custodian-media"
  comment     = "Fixed 1-year edge TTL for immutable-by-key media objects"
  min_ttl     = 31536000
  default_ttl = 31536000
  max_ttl     = 31536000

  parameters_in_cache_key_and_forwarded_to_origin {
    cookies_config {
      cookie_behavior = "none"
    }
    headers_config {
      header_behavior = "none"
    }
    query_strings_config {
      query_string_behavior = "none"
    }
  }
}

resource "aws_cloudfront_distribution" "media" {
  enabled         = true
  is_ipv6_enabled = true
  aliases         = [var.cdn_domain_name]
  price_class     = "PriceClass_100"
  comment         = "playground media CDN — private S3 media bucket via OAC"

  origin {
    origin_id                = "media"
    domain_name              = aws_s3_bucket.media.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.media.id
  }

  # Objects are served by key (cdn.<domain>/<key>), matching the extension-free
  # url custodian records and hands back. Read-only, cached long.
  default_cache_behavior {
    target_origin_id       = "media"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]

    cache_policy_id = aws_cloudfront_cache_policy.media.id
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.media_cdn.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }
}

# The bucket policy that lets only this distribution read the bucket: s3:GetObject
# to the CloudFront service principal, scoped by aws:SourceArn to this exact
# distribution. An OAC policy is not a public policy, so it coexists with the
# public-access block in buckets.tf (block_public_policy stays true).
data "aws_iam_policy_document" "media_cdn_read" {
  statement {
    sid       = "AllowMediaCDNRead"
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.media.arn}/*"]

    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }

    condition {
      test     = "StringEquals"
      variable = "aws:SourceArn"
      values   = [aws_cloudfront_distribution.media.arn]
    }
  }
}

resource "aws_s3_bucket_policy" "media_cdn_read" {
  bucket = aws_s3_bucket.media.id
  policy = data.aws_iam_policy_document.media_cdn_read.json
}

# Alias records point cdn.<domain> at the distribution: they resolve to both A and
# AAAA and cost nothing per query.
resource "aws_route53_record" "media_cdn_a" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = var.cdn_domain_name
  type    = "A"

  alias {
    name                   = aws_cloudfront_distribution.media.domain_name
    zone_id                = aws_cloudfront_distribution.media.hosted_zone_id
    evaluate_target_health = false
  }
}

resource "aws_route53_record" "media_cdn_aaaa" {
  zone_id = aws_route53_zone.playground.zone_id
  name    = var.cdn_domain_name
  type    = "AAAA"

  alias {
    name                   = aws_cloudfront_distribution.media.domain_name
    zone_id                = aws_cloudfront_distribution.media.hosted_zone_id
    evaluate_target_health = false
  }
}
