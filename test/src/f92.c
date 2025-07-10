struct Complex {
  long real;
  long imag;
};

long f(long x) {
  struct Complex nums[3];
  long sum;
  long i;
  
  nums[0].real = x;
  nums[0].imag = x + 1;
  nums[1].real = x * 2;
  nums[1].imag = x * 2 + 1;
  nums[2].real = x * 3;
  nums[2].imag = x * 3 + 1;
  
  sum = 0;
  for (i = 0; i < 3; i = i + 1) {
    sum = sum + nums[i].real + nums[i].imag;
  }
  
  return sum;
} 