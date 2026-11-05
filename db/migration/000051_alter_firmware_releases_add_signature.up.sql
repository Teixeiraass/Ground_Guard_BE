-- Base64 DER ECDSA P-256 signature over the image SHA-256. The controller
-- refuses any image whose signature doesn't verify with its embedded key.
ALTER TABLE "firmware_releases" ADD COLUMN "signature" text NOT NULL DEFAULT '';
