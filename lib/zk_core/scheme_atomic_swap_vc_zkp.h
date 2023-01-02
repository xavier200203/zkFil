#pragma once

#include "zkp.h"

namespace scheme::atomic_swap_vc {

inline uint64_t constexpr ZkpMimcCount() {
  return 1024;
}

struct ZkpItem {
  std::vector<ZkFr> o;
  std::vector<ZkFr> w;
  ZkFr seed;
  ZkFr seed_rand;
  ZkFr seed_mimc3_digest;
  ZkFr inner_product;
};

struct ZkvItem {
  std::vector<ZkFr> o;
  std::vector<ZkFr> w;
  ZkFr seed_mimc3_digest;
  ZkFr inner_product;
};

void GenerateZkProof(ZkProof& proof, ZkPk const& pk, ZkpItem const& item,
                     ZkVkPtr check_vk);

bool VerifyZkProof(ZkProof const& proof, ZkVk const& vk, ZkvItem const& item);
}  // namespace scheme::atomic_swap_vc