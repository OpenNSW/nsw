import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Button, Card, Heading, Text, Flex } from '@radix-ui/themes'
import { CheckIcon, Cross2Icon, ArrowLeftIcon } from '@radix-ui/react-icons'
import { useApi } from '../../services/ApiContext'

export function MockPaymentScreen() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const api = useApi()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const referenceNumber = searchParams.get('ref')

  const handleSimulatePayment = async (status: 'SUCCESS' | 'FAILED') => {
    if (!referenceNumber) {
      setError('No reference number found in URL.')
      return
    }

    try {
      setLoading(true)
      setError(null)
      await api.post('/payments/mock/callback', {
        reference_number: referenceNumber,
        status: status,
      })

      // Alert or simply navigate back
      alert(`Simulated ${status} for reference ${referenceNumber}`)

      // Navigate back to tasks list or a success page
      navigate(-1)
    } catch (err) {
      setError('Failed to simulate payment callback.')
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex justify-center items-center h-full p-6 bg-gray-50">
      <Card size="4" style={{ width: 450 }}>
        <Flex direction="column" gap="4">
          <Heading size="6" align="center">Payment Mock Checkout</Heading>

          <Text as="p" size="3" color="gray" align="center">
            You are in a development mock environment. No real funds will be charged.
          </Text>

          {referenceNumber ? (
            <div className="bg-gray-100 p-4 rounded text-center mb-2">
              <Text size="2" color="gray">Reference Number:</Text>
              <Heading size="4" mt="1">{referenceNumber}</Heading>
            </div>
          ) : (
            <Text color="red" align="center">Missing "ref" parameter in URL.</Text>
          )}

          {error && (
            <Text color="red" align="center">{error}</Text>
          )}

          <Flex gap="3" mt="2" direction="column">
            <Button
              size="3"
              color="green"
              variant="solid"
              disabled={!referenceNumber || loading}
              onClick={() => handleSimulatePayment('SUCCESS')}
            >
              <CheckIcon /> Simulate Success
            </Button>

            <Button
              size="3"
              color="red"
              variant="soft"
              disabled={!referenceNumber || loading}
              onClick={() => handleSimulatePayment('FAILED')}
            >
              <Cross2Icon /> Simulate Failure
            </Button>
          </Flex>

          <Flex justify="center" mt="4">
            <Button variant="ghost" color="gray" onClick={() => navigate(-1)} disabled={loading}>
              <ArrowLeftIcon /> Cancel and Return
            </Button>
          </Flex>
        </Flex>
      </Card>
    </div>
  )
}
