long f(long a, long b, long c, long d, long e, long f) {
  long result;
  long temp1, temp2, temp3, temp4;
  
  // 多くの計算を連続して行い、レジスタの使用を最適化
  temp1 = a + b;
  temp2 = c * d;
  temp3 = e - f;
  temp4 = temp1 + temp2;
  result = temp4 * temp3;
  
  temp1 = result + a;
  temp2 = temp1 * b;
  temp3 = temp2 - c;
  temp4 = temp3 / d;
  result = temp4 + e;
  
  return result;
} 